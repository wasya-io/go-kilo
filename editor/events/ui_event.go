package events

// UISubEventType はUI操作のサブタイプを表す
const (
	// 既存のイベントタイプ
	UIRefresh       SubEventType = "refresh"
	UIScroll        SubEventType = "scroll"
	UIStatusMessage SubEventType = "status_message"

	// 新規追加: エディタ領域の更新イベント
	UIEditorPartialRefresh SubEventType = "editor_partial_refresh" // 部分更新
	UICursorUpdate         SubEventType = "cursor_update"          // カーソル位置更新
	UISelectionUpdate      SubEventType = "selection_update"       // 選択範囲更新

	// 新規追加: コンポーネント固有の更新イベント
	UIStatusBarRefresh  SubEventType = "status_bar_refresh"  // ステータスバー更新
	UIMessageBarRefresh SubEventType = "message_bar_refresh" // メッセージバー更新

	// 新規追加: スクロール関連イベント
	UISmoothScroll  SubEventType = "smooth_scroll"  // スムーズスクロール
	UIPartialScroll SubEventType = "partial_scroll" // 部分スクロール
)

// UIEvent はUI操作イベントを表す
type UIEvent struct {
	BaseEvent
	SubType SubEventType
	Data    interface{} // イベントタイプに応じたデータ
}

// ScrollData はスクロールイベントのデータを表す
type ScrollData struct {
	RowOffset   int
	ColOffset   int
	IsSmooth    bool  // スムーズスクロール用
	ScrollLines []int // 部分スクロール用の行番号リスト
}

// EditorUpdateData はエディタ領域の更新データを表す
type EditorUpdateData struct {
	Lines    []int // 更新が必要な行番号のリスト
	ForceAll bool  // 全体更新が必要かどうか
}

// CursorData はカーソル位置の更新データを表す
type CursorData struct {
	X, Y      int
	IsVisible bool
}

// SelectionData は選択範囲の更新データを表す
type SelectionData struct {
	StartX, StartY int
	EndX, EndY     int
	IsActive       bool
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
