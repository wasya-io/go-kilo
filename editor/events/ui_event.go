package events

// UISubEventType はUI操作のサブタイプを表す
const (
	// UIRefresh は画面全体の更新
	UIRefresh SubEventType = "refresh"
	// UIScroll はスクロール操作
	UIScroll SubEventType = "scroll"
	// UIStatusMessage はステータスメッセージの更新
	UIStatusMessage SubEventType = "status_message"
)

// UIEvent はUI操作イベントを表す
type UIEvent struct {
	BaseEvent
	SubType SubEventType
	Data    interface{} // イベントタイプに応じたデータ
}

// ScrollData はスクロールイベントのデータを表す
type ScrollData struct {
	RowOffset int
	ColOffset int
}

// StatusMessageData はステータスメッセージのデータを表す
type StatusMessageData struct {
	Message string
	Args    []interface{}
}

// NewUIEvent は新しいUIEventを作成する
func NewUIEvent(subType SubEventType, data interface{}) *UIEvent {
	return &UIEvent{
		BaseEvent: NewBaseEvent(UIEventType),
		SubType:   subType,
		Data:      data,
	}
}
