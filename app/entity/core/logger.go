package core

type Logger interface {
	Log(messageType string, message string)
	ReadyWithType(messageType string) LogEntry
	Flush()
}

type LogEntry interface {
	WithType() LogEntry
	WithString() LogEntry
	WithInt() LogEntry
	Do(values ...interface{})
}
