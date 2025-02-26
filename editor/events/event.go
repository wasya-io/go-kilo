package events

import "time"

// EventType はイベントの種類を表す型
type EventType string

// イベントタイプの定義
const (
	SystemEventType EventType = "system"
	BufferEventType EventType = "buffer"
	UIEventType     EventType = "ui"
	FileEventType   EventType = "file"
	InputEventType  EventType = "input"
)

// イベントの優先度を表す定数
const (
	HighPriority   = 3
	MediumPriority = 2
	LowPriority    = 1
)

// Event は基本的なイベントのインターフェース
type Event interface {
	GetType() EventType
	GetTime() time.Time
	GetPriority() int
	HasError() bool
	GetError() error
	GetCurrentState() map[string]interface{}
}

// BaseEvent は基本的なイベントの実装
type BaseEvent struct {
	Type     EventType
	Time     time.Time
	Priority int
	Error    error
	State    map[string]interface{}
}

func (e *BaseEvent) GetType() EventType {
	return e.Type
}

func (e *BaseEvent) GetTime() time.Time {
	return e.Time
}

func (e *BaseEvent) GetPriority() int {
	return e.Priority
}

func (e *BaseEvent) HasError() bool {
	return e.Error != nil
}

func (e *BaseEvent) GetError() error {
	return e.Error
}

func (e *BaseEvent) GetCurrentState() map[string]interface{} {
	return e.State
}
