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
	// BufferRangeModified は範囲変更操作
	BufferRangeModified BufferOperationType = "range_modified"
)

// BufferEvent はバッファ操作イベントを表す
type BufferEvent struct {
	BaseEvent
	Operation BufferOperationType
	Position  Position    // 操作が行われた位置
	Data      interface{} // 操作に関連するデータ（文字、移動量など）
	Pre       BufferState // 操作前の状態
	Post      BufferState // 操作後の状態
	Range     *Range      // 変更された範囲（オプショナル）
}

// Position はバッファ内の位置を表す
type Position struct {
	X, Y int
}

// Range は変更された範囲を表す
type Range struct {
	Start Position
	End   Position
}

// BufferState はバッファの状態を表す
type BufferState struct {
	Content   string   // 影響を受けた行の内容
	IsDirty   bool     // 未保存の変更があるか
	CursorPos Position // カーソル位置
	Lines     []string // 影響を受けた行の範囲
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

// WithRange は変更範囲を設定したイベントを返す
func (e *BufferEvent) WithRange(start, end Position) *BufferEvent {
	e.Range = &Range{Start: start, End: end}
	return e
}

// HasChanges はバッファの状態が変更されたかどうかを返す
func (e *BufferEvent) HasChanges() bool {
	// コンテンツの比較
	if e.Pre.Content != e.Post.Content {
		return true
	}
	// ダーティフラグの比較
	if e.Pre.IsDirty != e.Post.IsDirty {
		return true
	}
	// カーソル位置の比較
	if e.Pre.CursorPos != e.Post.CursorPos {
		return true
	}
	// 行の内容の比較（スライスの比較）
	if !compareStringSlices(e.Pre.Lines, e.Post.Lines) {
		return true
	}
	return false
}

// compareStringSlices は2つの文字列スライスを比較する
func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
