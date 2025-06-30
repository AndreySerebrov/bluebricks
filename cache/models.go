package cache

//go:generate go tool mockgen  -destination=mock_fetcher.go -package=cache . Fetcher
type Fetcher interface {
	Fetch(id int) (string, error)
}

// go:generate mockgen -destination=mock_producer.go -package=tests -source=../../internal/processor/processor.go Producer
