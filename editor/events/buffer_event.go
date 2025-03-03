package events

import (
	"encoding/json"
	"errors"
	"time"
)

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
type BufferEventSubType string

const (
	BufferEventModify      BufferEventSubType = "modify"
	BufferEventSetState    BufferEventSubType = "set_state"
	BufferEventInsert      BufferEventSubType = "insert"
	BufferEventDelete      BufferEventSubType = "delete"
	BufferEventClear       BufferEventSubType = "clear"
	BufferContentChanged   BufferEventSubType = "content_changed"
	BufferStructuralChange BufferEventSubType = "structural_change"
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

// Position はバッファ内の位置を表す（編集操作の位置を示すために使用）
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
	Content string   // 影響を受けた行の内容
	IsDirty bool     // 未保存の変更があるか
	Lines   []string // 影響を受けた行の範囲
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
		BaseEvent: BaseEvent{
			Type:     BufferEventType,
			Time:     time.Now(),
			Priority: MediumPriority,
		},
		SubType: subType,
		Data:    data,
		changes: make([]BufferChangeData, 0),
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

// SetStates はバッファの状態を設定する
func (e *BufferEvent) SetStates(prev, curr BufferState) {
	// 状態をディープコピー
	e.prevState = BufferState{
		Content: prev.Content,
		IsDirty: prev.IsDirty,
		Lines:   append([]string(nil), prev.Lines...),
	}
	e.currState = BufferState{
		Content: curr.Content,
		IsDirty: curr.IsDirty,
		Lines:   append([]string(nil), curr.Lines...),
	}

	// 状態の変更を分析
	var change BufferChangeData
	switch e.SubType {
	case BufferEventSetState:
		// 完全な状態の置き換え
		change = BufferChangeData{
			ChangeType:   MultiLineEdit,
			IsStructural: true,
			StartLine:    0,
			EndLine:      len(curr.Lines) - 1,
		}
	case BufferEventModify, BufferContentChanged:
		// 内容の変更
		change = BufferChangeData{
			ChangeType:   SingleLineEdit,
			IsStructural: false,
		}
		if len(curr.Lines) > 0 {
			change.StartLine = 0
			change.EndLine = 0
		}
	}

	// 変更された行を特定
	if len(curr.Lines) > 0 || len(prev.Lines) > 0 {
		changedLines := make([]int, 0)
		maxLen := max(len(prev.Lines), len(curr.Lines))

		for i := 0; i < maxLen; i++ {
			var prevLine, currLine string
			if i < len(prev.Lines) {
				prevLine = prev.Lines[i]
			}
			if i < len(curr.Lines) {
				currLine = curr.Lines[i]
			}
			if prevLine != currLine {
				changedLines = append(changedLines, i)
			}
		}

		if len(changedLines) > 0 {
			change.AffectedLines = changedLines
			change.StartLine = changedLines[0]
			change.EndLine = changedLines[len(changedLines)-1]
		}

		// 構造的な変更の検出
		if len(prev.Lines) != len(curr.Lines) {
			change.IsStructural = true
			if len(curr.Lines) > len(prev.Lines) {
				change.ChangeType = LineInsert
			} else {
				change.ChangeType = LineDelete
			}
		}
	}

	e.changes = []BufferChangeData{change}
}

// max は2つの整数の大きい方を返す
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// determineSubType は操作タイプからサブタイプを決定する
func determineSubType(op BufferOperationType) BufferEventSubType {
	switch op {
	case BufferNewLine, BufferRangeModified:
		return BufferEventSetState
	default:
		return BufferEventModify
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

// BufferEventJSON はBufferEventのJSONシリアライズ用の構造体
type BufferEventJSON struct {
	Type      EventType              `json:"type"`
	Time      time.Time              `json:"time"`
	Priority  int                    `json:"priority"`
	Error     string                 `json:"error,omitempty"`
	State     map[string]interface{} `json:"state,omitempty"`
	SubType   BufferEventSubType     `json:"sub_type"`
	Data      interface{}            `json:"data"`
	Changes   []BufferChangeData     `json:"changes"`
	PrevState BufferState            `json:"prev_state"`
	CurrState BufferState            `json:"curr_state"`
}

// MarshalJSON はBufferEventをJSONに変換する
func (e *BufferEvent) MarshalJSON() ([]byte, error) {
	var errorStr string
	if e.Error != nil {
		errorStr = e.Error.Error()
	}

	return json.Marshal(BufferEventJSON{
		Type:      e.Type,
		Time:      e.Time,
		Priority:  e.Priority,
		Error:     errorStr,
		State:     e.State,
		SubType:   e.SubType,
		Data:      e.Data,
		Changes:   e.changes,
		PrevState: e.prevState,
		CurrState: e.currState,
	})
}

// UnmarshalJSON はJSONからBufferEventを復元する
func (e *BufferEvent) UnmarshalJSON(data []byte) error {
	var jsonEvent BufferEventJSON
	if err := json.Unmarshal(data, &jsonEvent); err != nil {
		return err
	}

	e.Type = jsonEvent.Type
	e.Time = jsonEvent.Time
	e.Priority = jsonEvent.Priority
	if jsonEvent.Error != "" {
		e.Error = errors.New(jsonEvent.Error)
	}
	e.State = jsonEvent.State
	e.SubType = jsonEvent.SubType
	e.Data = jsonEvent.Data
	e.changes = jsonEvent.Changes
	e.prevState = jsonEvent.PrevState
	e.currState = jsonEvent.CurrState

	return nil
}
