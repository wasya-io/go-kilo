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
	// BufferStateChange はバッファの状態変更操作
	BufferStateChange BufferOperationType = "state_change"
)

// BufferChangeType は変更の種類を表す
type BufferChangeType int

const (
	SingleLineEdit BufferChangeType = iota
	MultiLineEdit
	LineInsert
	LineDelete
	BlockOperation
)

// BufferEventSubType はバッファイベントのサブタイプを表す
type BufferEventSubType int

const (
	BufferContentChanged BufferEventSubType = iota
	BufferCursorMoved
	BufferStructuralChange
)

// BufferChangeData はバッファの変更情報を保持する
type BufferChangeData struct {
	AffectedLines []int
	ChangeType    BufferChangeType
	StartLine     int
	EndLine       int
	IsStructural  bool
	Operation     BufferOperationType
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

// BufferEvent はバッファの変更を表すイベント
type BufferEvent struct {
	BaseEvent
	SubType   BufferEventSubType
	Data      interface{}
	changes   []BufferChangeData
	prevState BufferState
	currState BufferState
}

// NewBufferEvent は新しいBufferEventを作成する
func NewBufferEvent(subType BufferEventSubType, data interface{}) *BufferEvent {
	return &BufferEvent{
		BaseEvent: BaseEvent{Type: BufferEventType},
		SubType:   subType,
		Data:      data,
		changes:   make([]BufferChangeData, 0),
	}
}

// NewBufferChangeEvent は変更情報を含むバッファイベントを作成する
func NewBufferChangeEvent(op BufferOperationType, pos Position, data interface{}, prevState, currState BufferState) *BufferEvent {
	event := &BufferEvent{
		BaseEvent: BaseEvent{Type: BufferEventType},
		SubType:   determineSubType(op),
		Data:      data,
		changes:   make([]BufferChangeData, 0),
		prevState: prevState,
		currState: currState,
	}

	// 変更データを追加
	change := BufferChangeData{
		ChangeType:   determineChangeType(op),
		IsStructural: isStructuralOperation(op),
		Operation:    op,
		StartLine:    pos.Y,
		EndLine:      pos.Y,
	}

	if prevState.Content != currState.Content {
		change.AffectedLines = []int{pos.Y}
	}

	event.AddChange(change)
	return event
}

// determineSubType は操作タイプからサブタイプを決定する
func determineSubType(op BufferOperationType) BufferEventSubType {
	switch op {
	case BufferMoveCursor:
		return BufferCursorMoved
	case BufferNewLine, BufferRangeModified:
		return BufferStructuralChange
	default:
		return BufferContentChanged
	}
}

// determineChangeType は操作タイプから変更タイプを決定する
func determineChangeType(op BufferOperationType) BufferChangeType {
	switch op {
	case BufferInsertChar, BufferDeleteChar:
		return SingleLineEdit
	case BufferNewLine:
		return LineInsert
	case BufferRangeModified:
		return BlockOperation
	default:
		return MultiLineEdit
	}
}

// isStructuralOperation は構造的な変更を伴う操作かどうかを判定する
func isStructuralOperation(op BufferOperationType) bool {
	switch op {
	case BufferNewLine, BufferRangeModified:
		return true
	default:
		return false
	}
}

// AddChange は変更情報を追加する
func (e *BufferEvent) AddChange(change BufferChangeData) {
	e.changes = append(e.changes, change)
}

// HasChanges は変更があるかどうかを返す
func (e *BufferEvent) HasChanges() bool {
	return len(e.changes) > 0
}

// GetChanges は変更情報のスライスを返す
func (e *BufferEvent) GetChanges() []BufferChangeData {
	return e.changes
}

// IsStructuralChange は構造的な変更があるかどうかを返す
func (e *BufferEvent) IsStructuralChange() bool {
	for _, change := range e.changes {
		if change.IsStructural {
			return true
		}
	}
	return false
}

// GetStates は前回と現在の状態を返す
func (e *BufferEvent) GetStates() (BufferState, BufferState) {
	return e.prevState, e.currState
}

// GetOperation は最初の変更の操作タイプを返す
func (e *BufferEvent) GetOperation() BufferOperationType {
	if len(e.changes) > 0 {
		return e.changes[0].Operation
	}
	return ""
}

// GetAffectedLines は影響を受けた行の一覧を返す
func (e *BufferEvent) GetAffectedLines() []int {
	lines := make(map[int]bool)
	for _, change := range e.changes {
		for _, line := range change.AffectedLines {
			lines[line] = true
		}
	}

	result := make([]int, 0, len(lines))
	for line := range lines {
		result = append(result, line)
	}
	return result
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
