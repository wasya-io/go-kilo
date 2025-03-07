package core

type Logger interface {
	Log(messageType string, message string)
	Flush()
}
