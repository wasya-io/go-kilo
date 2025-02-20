package events

// Event は全てのイベントの基本インターフェース
type Event interface {
	GetType() EventType
	// エラーハンドリング用のメソッドを追加
	HasError() bool
	GetError() error
	SetError(err error)
	// リカバリー用のメソッドを追加
	GetPreviousState() interface{}
	GetCurrentState() interface{}
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
	Type      EventType
	err       error
	prevState interface{}
	currState interface{}
}

func (e BaseEvent) GetType() EventType {
	return e.Type
}

func (e *BaseEvent) HasError() bool {
	return e.err != nil
}

func (e *BaseEvent) GetError() error {
	return e.err
}

func (e *BaseEvent) SetError(err error) {
	e.err = err
}

func (e *BaseEvent) GetPreviousState() interface{} {
	return e.prevState
}

func (e *BaseEvent) GetCurrentState() interface{} {
	return e.currState
}

func (e *BaseEvent) SetStates(prev, curr interface{}) {
	e.prevState = prev
	e.currState = curr
}
