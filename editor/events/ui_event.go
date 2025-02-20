package events

// UIEventSubType はUIイベントのサブタイプを表す
type UIEventSubType int

const (
	UIRefresh UIEventSubType = iota
	UIScroll
	UIStatusMessage
	UIEditorPartialRefresh
	UICursorUpdate
	UIStatusBarRefresh
	UIMessageBarRefresh
)

// ScrollData はスクロールイベントのデータを表す
type ScrollData struct {
	ColOffset int
	RowOffset int
	IsSmooth  bool
}

// StatusMessageData はステータスメッセージイベントのデータを表す
type StatusMessageData struct {
	Message string
	Args    []interface{}
}

// EditorUpdateData はエディタ更新イベントのデータを表す
type EditorUpdateData struct {
	Lines    []int
	ForceAll bool
}

// UIEvent はUI更新イベントを表す
type UIEvent struct {
	BaseEvent
	SubType UIEventSubType
	Data    interface{}
}

// NewUIEvent は新しいUIEventを作成する
func NewUIEvent(subType UIEventSubType, data interface{}) *UIEvent {
	return &UIEvent{
		BaseEvent: BaseEvent{Type: UIEventType},
		SubType:   subType,
		Data:      data,
	}
}
