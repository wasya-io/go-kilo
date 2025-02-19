package events

import "time"

// Event はすべてのイベントの基本インターフェース
type Event interface {
	// Type はイベントの種類を返す
	Type() EventType
	// Timestamp はイベントが発生した時刻を返す
	Timestamp() time.Time
}

// EventType はイベントの種類を表す型
type EventType string

// 基本的なイベントタイプの定義
const (
	// InputEventType は入力関連のイベント
	InputEventType EventType = "input"
	// BufferEventType はバッファ操作関連のイベント
	BufferEventType EventType = "buffer"
	// UIEventType はUI更新関連のイベント
	UIEventType EventType = "ui"
	// FileEventType はファイル操作関連のイベント
	FileEventType EventType = "file"
)

// SubEventType はイベントのサブタイプを表す型
type SubEventType string

// BaseEvent は基本的なイベント情報を持つ構造体
type BaseEvent struct {
	eventType EventType
	timestamp time.Time
}

// NewBaseEvent は新しいBaseEventを作成する
func NewBaseEvent(eventType EventType) BaseEvent {
	return BaseEvent{
		eventType: eventType,
		timestamp: time.Now(),
	}
}

// Type はイベントの種類を返す
func (e BaseEvent) Type() EventType {
	return e.eventType
}

// Timestamp はイベントが発生した時刻を返す
func (e BaseEvent) Timestamp() time.Time {
	return e.timestamp
}
