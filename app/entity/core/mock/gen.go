package mock_core

//go:generate mockgen -destination=mock_logger.go -package=mock_core github.com/wasya-io/go-kilo/app/entity/core Logger,LogEntry
