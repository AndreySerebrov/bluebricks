package cache

import "go.uber.org/zap"

type loggerZap struct {
	base      Fetcher
	zapLogger *zap.Logger
}

func newLogger(fetcher Fetcher) *loggerZap {
	return &loggerZap{
		base:      fetcher,
		zapLogger: zap.NewExample(),
	}
}

func (l *loggerZap) Fetch(id int) (string, error) {
	data, err := l.base.Fetch(id)
	if err != nil {
		l.zapLogger.Error("Fetch failed", zap.Int("id", id), zap.Error(err))
		return "", err
	}
	return data, nil
}
