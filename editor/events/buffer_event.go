package events

// BufferOperationType はバッファ操作の種類を表す
type BufferOperationType string

const (
	// BufferLoadContent は内容のロード操作
	BufferLoadContent BufferOperationType = "load_content"
	// BufferInsertChar は文字挿入操作
	BufferInsertChar BufferOperationType = "insert_char"
	// BufferDeleteChar は文字削除操作
	BufferDeleteChar BufferOperationType = "delete_char"
	// BufferNewLine は改行挿入操作
	BufferNewLine BufferOperationType = "new_line"
	// BufferMoveCursor はカーソル移動操作
	BufferMoveCursor BufferOperationType = "move_cursor"
)

// BufferEvent はバッファ操作イベントを表す
type BufferEvent struct {
	BaseEvent
	Operation BufferOperationType
	Position  Position    // 操作が行われた位置
	Data      interface{} // 操作に関連するデータ（文字、移動量など）
	Pre       BufferState // 操作前の状態
	Post      BufferState // 操作後の状態
}

// Position はバッファ内の位置を表す
type Position struct {
	X, Y int
}

// BufferState はバッファの状態を表す
type BufferState struct {
	Content   string   // 影響を受けた行の内容
	IsDirty   bool     // 未保存の変更があるか
	CursorPos Position // カーソル位置
}

// NewBufferEvent は新しいBufferEventを作成する
func NewBufferEvent(op BufferOperationType, pos Position, data interface{}, pre, post BufferState) *BufferEvent {
	return &BufferEvent{
		BaseEvent: NewBaseEvent(BufferEventType),
		Operation: op,
		Position:  pos,
		Data:      data,
		Pre:       pre,
		Post:      post,
	}
}
