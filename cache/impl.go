package cache

import (
	"container/list"
	"sync"
	"time"
)

type note struct {
	ts time.Time
	*sync.Mutex
	val string
	id  *int
}

type impl struct {
	list        *list.List
	valMap      sync.Map
	fetcher     Fetcher
	globalMutex *sync.Mutex
	cond        *sync.Cond
	cap         int
}

func New(fetcher Fetcher, interval time.Duration, cap int) *impl {
	instance := &impl{
		list:        list.New(),
		valMap:      sync.Map{},
		fetcher:     fetcher,
		globalMutex: &sync.Mutex{},
		cond:        sync.NewCond(&sync.Mutex{}),
		cap:         cap,
	}
	instance.backgroundCleanup(interval)
	return instance
}

func (c *impl) Fetch(id int) (string, error) {

	element, ok := c.valMap.Load(id)
	if ok {
		n := element.(*list.Element).Value.(*note)
		if n.id != nil {
			dataToReturn := n.val
			n.Lock()
			n.ts = time.Now()
			n.Unlock()

			c.globalMutex.Lock()
			c.list.MoveToFront(element.(*list.Element))
			c.globalMutex.Unlock()
			return dataToReturn, nil
		}
	}
	var el *list.Element

	// Global critical section to ensure that only one goroutine
	// can create a new note for the same id at a time.
	c.globalMutex.Lock()
	if element, ok = c.valMap.Load(id); !ok {
		// If the element is not found, we create a new one
		// and add it to the front of the list.
		// If the list is too long, we remove the oldest element.
		if c.list.Len() >= c.cap {
			// Remove the oldest element if we have reached the limit
			oldest := c.list.Back()
			if oldest != nil {
				c.list.Remove(oldest)
				n := oldest.Value.(*note)
				c.valMap.Delete(*n.id)
			}
		}
		// Create a new note and add it to the front of the list
		el = c.list.PushFront(&note{
			ts:    time.Now(),
			Mutex: &sync.Mutex{},
		})
		c.valMap.Store(id, el)
		// If the list was empty, signal the condition variable
		if c.list.Len() == 1 {
			c.cond.Signal()
		}
		// The element had been created by another goroutine
	} else {
		el = element.(*list.Element)
	}
	c.globalMutex.Unlock()

	note := el.Value.(*note)
	note.Mutex.Lock()
	// Lock the note to ensure that no other goroutine can modify it
	// while we are fetching the data.
	defer note.Mutex.Unlock()
	if note.id != nil {
		return note.val, nil
	}
	data, err := c.fetcher.Fetch(id)
	if err != nil {
		c.globalMutex.Lock()
		c.list.Remove(el)
		c.valMap.Delete(id)
		c.globalMutex.Unlock()
		return "", err
	}
	note.val = data
	note.id = &id
	note.ts = time.Now()

	return data, nil
}

func (c *impl) backgroundCleanup(interval time.Duration) {

	go func() {
		for {
			c.globalMutex.Lock()
			var ts time.Time
			for e := c.list.Back(); e != nil; e = e.Prev() {
				n := e.Value.(*note)
				if time.Since(n.ts) > interval && n.id != nil {
					c.list.Remove(e)
					c.valMap.Delete(*n.id)
					continue
				}
				ts = e.Value.(*note).ts
				ts = ts.Add(interval)

			}
			c.globalMutex.Unlock()
			// List is not empty, wait for the next cleanup interval
			if !ts.IsZero() {
				time.Sleep(time.Until(ts))
			} else {
				// List is empty, wait for a signal to wake up
				c.cond.L.Lock()
				c.cond.Wait()
			}
		}
	}()
}
