package editor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wasya-io/go-kilo/editor/events"
)

type Event = events.Event
type BufferEvent = events.BufferEvent
type BaseEvent = events.BaseEvent
type SystemEvent = events.SystemEvent
type UIEvent = events.UIEvent
type FileEvent = events.FileEvent
type InputEvent = events.InputEvent
type EventType = events.EventType
type SystemEventSubType = events.SystemEventSubType
type SaveEvent = events.SaveEvent
type QuitEvent = events.QuitEvent
type StatusEvent = events.StatusEvent
type BufferChangeData = events.BufferChangeData
type BufferState = events.BufferState

const (
	BufferEventType = events.BufferEventType
	SystemEventType = events.SystemEventType
	UIEventType     = events.UIEventType
	FileEventType   = events.FileEventType
	InputEventType  = events.InputEventType
	SystemSave      = events.SystemSave
	SystemQuit      = events.SystemQuit
	SystemStatus    = events.SystemStatus
)

// EventPublisher はイベントの発行を担当するインターフェース
type EventPublisher interface {
	BeginBatch()
	Publish(event events.Event) error
	EndBatch()
}

// StateStorage は状態の保存を担当するインターフェース
type StateStorage interface {
	SaveState(state EditorState) error
	LoadState(timestamp time.Time) (EditorState, error)
}

// StateRestorer は状態の復元を担当するインターフェース
type StateRestorer interface {
	RestoreState(state EditorState) error
}

// EditorState はエディタの状態を表す構造体
type EditorState struct {
	Timestamp time.Time
	Events    []events.Event
	// 将来的な拡張のためのフィールド
	// BufferState  BufferState
	// CursorState  CursorState
}

// MarshalJSON はEditorStateをJSONに変換する
func (s EditorState) MarshalJSON() ([]byte, error) {
	type EventWithMetadata struct {
		Event     Event  `json:"event"`
		Checksum  string `json:"checksum"`
		Timestamp int64  `json:"timestamp"`
	}

	// イベントにメタデータを付加
	events := make([]EventWithMetadata, 0, len(s.Events))
	for _, event := range s.Events {
		if event == nil {
			continue
		}
		// イベントのメタデータを生成
		var checksum string
		if bufferEvent, ok := event.(*BufferEvent); ok {
			prevState, currState := bufferEvent.GetStates()
			checksum = fmt.Sprintf("%v_%v_%v_%v_%v",
				event.GetTime().UnixNano(),
				event.GetType(),
				bufferEvent.SubType,
				prevState.Content,
				currState.Content,
			)
		} else {
			checksum = fmt.Sprintf("%v_%v",
				event.GetTime().UnixNano(),
				event.GetType(),
			)
		}

		events = append(events, EventWithMetadata{
			Event:     event,
			Checksum:  checksum,
			Timestamp: event.GetTime().UnixNano(),
		})
	}

	return json.Marshal(struct {
		Timestamp int64               `json:"timestamp"`
		Events    []EventWithMetadata `json:"events"`
	}{
		Timestamp: s.Timestamp.UnixNano(),
		Events:    events,
	})
}

// createEvent は適切な型のイベントを作成する
func createEvent(eventType EventType) Event {
	switch eventType {
	case BufferEventType:
		return &events.BufferEvent{}
	case SystemEventType:
		return &events.BaseSystemEvent{
			BaseEvent: events.BaseEvent{
				Type: events.SystemEventType,
				Time: time.Now(),
			},
			SubType: events.SystemStatus, // デフォルトのサブタイプ
		}
	case UIEventType:
		return &events.UIEvent{}
	case FileEventType:
		return &events.FileEvent{}
	case InputEventType:
		return &events.InputEvent{}
	default:
		return nil
	}
}

// UnmarshalJSON はJSONからEditorStateを復元する
func (s *EditorState) UnmarshalJSON(data []byte) error {
	type EventWithMetadata struct {
		Event     json.RawMessage `json:"event"`
		Checksum  string          `json:"checksum"`
		Timestamp int64           `json:"timestamp"`
	}

	var jsonState struct {
		Timestamp int64               `json:"timestamp"`
		Events    []EventWithMetadata `json:"events"`
	}

	if err := json.Unmarshal(data, &jsonState); err != nil {
		return err
	}

	s.Timestamp = time.Unix(0, jsonState.Timestamp)
	s.Events = make([]events.Event, 0, len(jsonState.Events))

	seen := make(map[string]bool)

	for _, eventData := range jsonState.Events {
		// 重複チェック
		if seen[eventData.Checksum] {
			continue
		}
		seen[eventData.Checksum] = true

		// イベントの型を判別
		var eventType struct {
			Type events.EventType `json:"type"`
		}
		if err := json.Unmarshal(eventData.Event, &eventType); err != nil {
			return err
		}

		event := createEvent(eventType.Type)
		if event == nil {
			fmt.Printf("Warning: Skipping unknown event type: %v\n", eventType.Type)
			continue
		}

		if err := json.Unmarshal(eventData.Event, event); err != nil {
			fmt.Printf("Warning: Failed to unmarshal event: %v\n", err)
			continue
		}

		// チェックサムの検証
		var computedChecksum string
		if bufferEvent, ok := event.(*events.BufferEvent); ok {
			prevState, currState := bufferEvent.GetStates()
			computedChecksum = fmt.Sprintf("%v_%v_%v_%v_%v",
				eventData.Timestamp,
				eventType.Type,
				bufferEvent.SubType,
				prevState.Content,
				currState.Content,
			)
		} else {
			computedChecksum = fmt.Sprintf("%v_%v",
				eventData.Timestamp,
				eventType.Type,
			)
		}

		if computedChecksum == eventData.Checksum {
			s.Events = append(s.Events, event)
		}
	}

	return nil
}

// Verify that EditorStateManager implements StateRecoveryRequester
var _ events.StateRecoveryRequester = (*EditorStateManager)(nil)

// EditorStateManager はエディタの状態を管理する
type EditorStateManager struct {
	states         []EditorState
	maxStates      int
	eventPublisher EventPublisher
	stateStorage   StateStorage
	stateRestorer  StateRestorer
}

// NewEditorStateManager は新しいEditorStateManagerを作成する
func NewEditorStateManager(publisher EventPublisher) *EditorStateManager {
	return &EditorStateManager{
		states:         make([]EditorState, 0),
		maxStates:      10,
		eventPublisher: publisher,
	}
}

// SetStateStorage は状態の保存方法を設定する
func (sm *EditorStateManager) SetStateStorage(storage StateStorage) {
	sm.stateStorage = storage
}

// SetStateRestorer は状態の復元方法を設定する
func (sm *EditorStateManager) SetStateRestorer(restorer StateRestorer) {
	sm.stateRestorer = restorer
}

// CreateSnapshot は現在の状態のスナップショットを作成する
func (sm *EditorStateManager) CreateSnapshot(events []Event) EditorState {
	if len(events) == 0 {
		fmt.Printf("Warning: Attempting to create snapshot with no events\n")
		return EditorState{}
	}

	// イベントの重複を防ぐためのマップ
	seen := make(map[string]bool)
	uniqueEvents := make([]Event, 0, len(events))

	for _, event := range events {
		if event == nil {
			fmt.Printf("Warning: Skipping nil event in CreateSnapshot\n")
			continue
		}

		// イベントの一意性を確認するためのキーを生成
		key := fmt.Sprintf("%v_%v_%v", event.GetType(), event.GetTime().UnixNano(), event.GetPriority())
		if bufferEvent, ok := event.(*BufferEvent); ok {
			prevState, currState := bufferEvent.GetStates()
			key += fmt.Sprintf("_%v_%v_%v", bufferEvent.SubType, prevState.Content, currState.Content)
		}

		// 重複していないイベントのみを追加
		if !seen[key] {
			seen[key] = true
			uniqueEvents = append(uniqueEvents, copyEvent(event))
		}
	}

	if len(uniqueEvents) == 0 {
		fmt.Printf("Warning: No valid events to save after deduplication\n")
		return EditorState{}
	}

	state := EditorState{
		Timestamp: time.Now(),
		Events:    uniqueEvents,
	}

	fmt.Printf("Debug: Creating snapshot with %d events at %v\n", len(uniqueEvents), state.Timestamp)

	// スナップショットを正しい時系列で保存
	insertIndex := 0
	for i := len(sm.states) - 1; i >= 0; i-- {
		if sm.states[i].Timestamp.Before(state.Timestamp) {
			insertIndex = i + 1
			break
		}
	}

	// 新しい状態を挿入
	if insertIndex == len(sm.states) {
		sm.states = append(sm.states, state)
	} else {
		sm.states = append(sm.states[:insertIndex], append([]EditorState{state}, sm.states[insertIndex:]...)...)
	}

	// 状態数の制限
	if len(sm.states) > sm.maxStates {
		sm.states = sm.states[len(sm.states)-sm.maxStates:]
	}

	fmt.Printf("Debug: Total states in memory: %d (latest state has %d events)\n",
		len(sm.states), len(sm.states[len(sm.states)-1].Events))

	return state
}

// persistAndVerifyState は状態を永続化し、その整合性を検証する
func (sm *EditorStateManager) persistAndVerifyState(state EditorState) error {
	if err := sm.stateStorage.SaveState(state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// 保存した状態を即座に読み込んで検証
	loaded, err := sm.stateStorage.LoadState(state.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to verify saved state: %w", err)
	}

	fmt.Printf("Debug: Verified saved state: found %d events\n", len(loaded.Events))
	if len(loaded.Events) != len(state.Events) {
		return fmt.Errorf("event count mismatch after save/load: saved=%d, loaded=%d",
			len(state.Events), len(loaded.Events))
	}

	return nil
}

// copyEvent はイベントの深いコピーを作成する
func copyEvent(event Event) Event {
	switch e := event.(type) {
	case *BufferEvent:
		// BufferEventの完全なコピーを作成
		prevState, currState := e.GetStates()
		bufferEvent := &BufferEvent{
			BaseEvent: BaseEvent{
				Type:     e.Type,
				Time:     e.Time,
				Priority: e.Priority,
				Error:    e.Error,
				State:    copyMap(e.State),
			},
			SubType: e.SubType,
			Data:    e.Data,
		}

		// 変更情報をコピー
		changes := e.GetChanges()
		for _, change := range changes {
			bufferEvent.AddChange(BufferChangeData{
				AffectedLines: append([]int(nil), change.AffectedLines...),
				ChangeType:    change.ChangeType,
				StartLine:     change.StartLine,
				EndLine:       change.EndLine,
				IsStructural:  change.IsStructural,
				Operation:     change.Operation,
			})
		}

		// 状態情報をコピー
		bufferEvent.SetStates(
			BufferState{
				Content: prevState.Content,
				IsDirty: prevState.IsDirty,
				Lines:   append([]string(nil), prevState.Lines...),
			},
			BufferState{
				Content: currState.Content,
				IsDirty: currState.IsDirty,
				Lines:   append([]string(nil), currState.Lines...),
			},
		)
		return bufferEvent

	default:
		// その他のイベントタイプはそのまま返す
		return event
	}
}

// copyMap はマップのディープコピーを作成するヘルパー関数
func copyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// RecoverFromSnapshot は指定されたタイムスタンプのスナップショットから状態を復元する
func (sm *EditorStateManager) RecoverFromSnapshot(timestamp time.Time) error {
	if len(sm.states) == 0 {
		return fmt.Errorf("no states available for recovery")
	}

	// 指定された時点より前の最新のスナップショットを探す
	var targetState EditorState
	found := false
	foundIndex := -1

	fmt.Printf("Debug: Searching for state before %v\n", timestamp)
	for i := 0; i < len(sm.states); i++ {
		state := sm.states[i]
		if state.Timestamp.Before(timestamp) || state.Timestamp.Equal(timestamp) {
			targetState = state
			found = true
			foundIndex = i
			fmt.Printf("Debug: Found candidate state at index %d with timestamp %v (%d events)\n",
				i, state.Timestamp, len(state.Events))
		} else {
			// より新しい状態が見つかったら探索終了
			break
		}
	}

	if !found {
		return fmt.Errorf("no suitable state found for timestamp %v", timestamp)
	}

	fmt.Printf("Debug: Selected state at index %d for restoration (contains %d events)\n",
		foundIndex, len(targetState.Events))

	// 保存された状態から復元
	if err := sm.restoreState(targetState); err != nil {
		return fmt.Errorf("failed to restore state: %w", err)
	}

	// 現在の状態として設定（復元された状態より新しい状態は削除）
	sm.states = sm.states[:foundIndex+1]
	return nil
}

// restoreState は指定された状態を復元する
func (sm *EditorStateManager) restoreState(state EditorState) error {
	if len(state.Events) == 0 {
		return fmt.Errorf("cannot restore state with no events")
	}

	fmt.Printf("Debug: Starting state restoration with %d events\n", len(state.Events))

	// カスタムのリストアラーが設定されている場合はそれを使用
	if sm.stateRestorer != nil {
		return sm.stateRestorer.RestoreState(state)
	}

	// デフォルトの復元処理
	if sm.eventPublisher == nil {
		return fmt.Errorf("no event publisher configured")
	}

	// 最後のバッファ状態イベントを探す
	var targetState events.BufferState
	var foundValidState bool

	for i := len(state.Events) - 1; i >= 0; i-- {
		event := state.Events[i]
		if bufferEvent, ok := event.(*BufferEvent); ok {
			_, currState := bufferEvent.GetStates()
			if len(currState.Lines) > 0 || currState.Content != "" {
				targetState = currState
				foundValidState = true
				break
			}
		}
	}

	if !foundValidState {
		return fmt.Errorf("no valid buffer state found in events")
	}

	fmt.Printf("Debug: Target state constructed: Content=%q, Lines=%q\n",
		targetState.Content, targetState.Lines)

	// バッファを直接リセットするイベントを発行
	resetEvent := events.NewBufferEvent(events.BufferEventSetState, targetState)
	resetEvent.SetStates(
		events.BufferState{Content: "", IsDirty: false, Lines: []string{""}},
		targetState,
	)

	// イベントを単一のバッチで発行
	sm.eventPublisher.BeginBatch()
	if err := sm.eventPublisher.Publish(resetEvent); err != nil {
		sm.eventPublisher.EndBatch()
		return fmt.Errorf("failed to reset state: %w", err)
	}
	sm.eventPublisher.EndBatch()

	fmt.Printf("Debug: State restoration completed with target state\n")
	return nil
}

// RequestStateRecovery は状態の復元要求を処理する
func (sm *EditorStateManager) RequestStateRecovery(timestamp time.Time) error {
	return sm.RecoverFromSnapshot(timestamp)
}

// GetStates は状態の一覧を返す
func (sm *EditorStateManager) GetStates() []EditorState {
	return append([]EditorState{}, sm.states...)
}

// GetLatestState は最新の状態を返す
func (sm *EditorStateManager) GetLatestState() (EditorState, error) {
	if len(sm.states) == 0 {
		return EditorState{}, fmt.Errorf("no states available")
	}
	return sm.states[len(sm.states)-1], nil
}

// FileBasedStateStorage は状態をファイルシステムに保存するストレージ実装
type FileBasedStateStorage struct {
	baseDir string
}

// NewFileBasedStateStorage は新しいファイルベースのストレージを作成する
func NewFileBasedStateStorage(baseDir string) *FileBasedStateStorage {
	return &FileBasedStateStorage{
		baseDir: baseDir,
	}
}

// SaveState は状態をJSONファイルとして保存する
func (s *FileBasedStateStorage) SaveState(state EditorState) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// イベントが空の場合は保存しない
	if len(state.Events) == 0 {
		return fmt.Errorf("cannot save state with no events")
	}

	filename := filepath.Join(s.baseDir, fmt.Sprintf("snapshot_%d.json", state.Timestamp.UnixNano()))

	// 一時ファイルに書き込み
	tempFile := filename + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(state); err != nil {
		os.Remove(tempFile) // エラー時は一時ファイルを削除
		return fmt.Errorf("failed to encode state: %w", err)
	}

	// ファイルを閉じて、本来のファイル名に改名
	file.Close()
	if err := os.Rename(tempFile, filename); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to save state file: %w", err)
	}

	return nil
}

// LoadState は指定時刻以前の最新の状態を読み込む
func (s *FileBasedStateStorage) LoadState(timestamp time.Time) (EditorState, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return EditorState{}, fmt.Errorf("failed to read directory: %w", err)
	}

	fmt.Printf("Debug: Looking for state before %v in %s\n", timestamp, s.baseDir)
	fmt.Printf("Debug: Found %d files in directory\n", len(entries))

	var matchingStates []EditorState
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) == ".tmp" {
			continue
		}

		filePath := filepath.Join(s.baseDir, entry.Name())
		fmt.Printf("Debug: Processing file: %s\n", filePath)

		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Warning: Failed to read file %s: %v\n", entry.Name(), err)
			continue
		}

		var tempState EditorState
		if err := json.Unmarshal(fileContent, &tempState); err != nil {
			fmt.Printf("Warning: Failed to decode state file %s: %v\n", entry.Name(), err)
			continue
		}

		// イベントの検証
		if len(tempState.Events) == 0 {
			fmt.Printf("Warning: No events found in state file %s\n", entry.Name())
			continue
		}

		for i, event := range tempState.Events {
			if event == nil {
				fmt.Printf("Warning: Found nil event at index %d in file %s\n", i, entry.Name())
				continue
			}
		}

		if tempState.Timestamp.Before(timestamp) || tempState.Timestamp.Equal(timestamp) {
			fmt.Printf("Debug: Found valid state in %s with %d events at %v\n",
				entry.Name(), len(tempState.Events), tempState.Timestamp)
			matchingStates = append(matchingStates, tempState)
		}
	}

	if len(matchingStates) == 0 {
		return EditorState{}, fmt.Errorf("no snapshot found before %v", timestamp)
	}

	// イベントの数が最も多い状態を選択
	var latestState EditorState
	var maxEvents int
	for _, state := range matchingStates {
		if len(state.Events) > maxEvents {
			latestState = state
			maxEvents = len(latestState.Events)
		}
	}

	fmt.Printf("Debug: Selected state with timestamp %v containing %d events\n",
		latestState.Timestamp, len(latestState.Events))

	for i, event := range latestState.Events {
		fmt.Printf("Debug: Event[%d] Type=%v\n", i, event.GetType())
	}

	return latestState, nil
}

// Cleanup はストレージのデータを削除する
func (s *FileBasedStateStorage) Cleanup() error {
	return os.RemoveAll(s.baseDir)
}
