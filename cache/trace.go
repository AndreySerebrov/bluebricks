package cache

import (
	opentracing "github.com/opentracing/opentracing-go"
)

type trace struct {
	base Fetcher
}

func newTrace(fetcher Fetcher) *trace {
	return &trace{
		base: fetcher,
	}
}

func (t *trace) Fetch(id int) (string, error) {
	span := opentracing.StartSpan("Fetch")
	var err error
	var data string
	defer func() {
		span.LogKV("event", "fetch_complete", "id", id, "error", err)
		span.Finish()
	}()
	data, err = t.base.Fetch(id)
	if err != nil {
		return "", err
	}
	return data, nil
}
