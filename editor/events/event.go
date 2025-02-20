package events

// Event は全てのイベントの基本インターフェース
type Event interface {
	GetType() EventType
}

// EventType はイベントの種類を表す
type EventType int

const (
	BufferEventType EventType = iota
	UIEventType
	FileEventType
	InputEventType
)

// BaseEvent は全てのイベントの基本構造体
type BaseEvent struct {
	Type EventType
}

func (e BaseEvent) GetType() EventType {
	return e.Type
}
