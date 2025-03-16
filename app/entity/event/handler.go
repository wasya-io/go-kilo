package event

// Handler はイベントを処理するためのインターフェースです。
type Handler interface {
	// HandleEvent はイベントを処理します。
	// 処理が成功したかどうかとエラーを返します。
	HandleEvent(event Event) (bool, error)

	// GetHandledEventTypes は、このハンドラーが処理できるイベントタイプのリストを返します。
	GetHandledEventTypes() []EventType
}

// HandlerFunc は Handler インターフェースを実装する関数型です。
type HandlerFunc func(event Event) (bool, error)

// HandleEvent は HandlerFunc 型が Handler インターフェースを満たすための実装です。
func (f HandlerFunc) HandleEvent(event Event) (bool, error) {
	return f(event)
}

// SingleTypeHandler は1つのイベントタイプだけを処理するハンドラーです。
type SingleTypeHandler struct {
	EventType EventType
	Handler   HandlerFunc
}

// HandleEvent はイベントを処理します。
func (h *SingleTypeHandler) HandleEvent(event Event) (bool, error) {
	if event.Type == h.EventType {
		return h.Handler(event)
	}
	return false, nil
}

// GetHandledEventTypes は、このハンドラーが処理できるイベントタイプのリストを返します。
func (h *SingleTypeHandler) GetHandledEventTypes() []EventType {
	return []EventType{h.EventType}
}

// NewSingleTypeHandler は新しい単一タイプのハンドラーを作成します。
func NewSingleTypeHandler(eventType EventType, handler HandlerFunc) *SingleTypeHandler {
	return &SingleTypeHandler{
		EventType: eventType,
		Handler:   handler,
	}
}
