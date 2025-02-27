package editor

import (
	"fmt"
	"time"

	"github.com/wasya-io/go-kilo/editor/events"
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
func (sm *EditorStateManager) CreateSnapshot(events []events.Event) {
	state := EditorState{
		Timestamp: time.Now(),
		Events:    events,
	}

	sm.states = append(sm.states, state)
	if len(sm.states) > sm.maxStates {
		sm.states = sm.states[1:]
	}

	// 永続化が設定されている場合は保存を試みる
	if sm.stateStorage != nil {
		if err := sm.stateStorage.SaveState(state); err != nil {
			// TODO: エラーログの記録
			fmt.Printf("Failed to save state: %v\n", err)
		}
	}
}

// RecoverFromSnapshot は指定された時点のスナップショットから復元する
func (sm *EditorStateManager) RecoverFromSnapshot(timestamp time.Time) error {
	// まず永続化ストレージから状態の復元を試みる
	if sm.stateStorage != nil {
		if state, err := sm.stateStorage.LoadState(timestamp); err == nil {
			return sm.restoreState(state)
		}
	}

	// メモリ内の状態から復元を試みる
	for i := len(sm.states) - 1; i >= 0; i-- {
		if sm.states[i].Timestamp.Before(timestamp) || sm.states[i].Timestamp.Equal(timestamp) {
			return sm.restoreState(sm.states[i])
		}
	}

	return fmt.Errorf("no snapshot found before %v", timestamp)
}

// restoreState は指定された状態を復元する
func (sm *EditorStateManager) restoreState(state EditorState) error {
	// カスタムのリストアラーが設定されている場合はそれを使用
	if sm.stateRestorer != nil {
		return sm.stateRestorer.RestoreState(state)
	}

	// デフォルトの復元処理（イベントの再生）
	if sm.eventPublisher != nil {
		sm.eventPublisher.BeginBatch()
		for _, event := range state.Events {
			sm.eventPublisher.Publish(event)
		}
		sm.eventPublisher.EndBatch()
		return nil
	}

	return fmt.Errorf("no event publisher or state restorer configured")
}

// RequestStateRecovery は状態の復元要求を処理する
func (sm *EditorStateManager) RequestStateRecovery(timestamp time.Time) error {
	return sm.RecoverFromSnapshot(timestamp)
}

// GetStates は状態の一覧を返す
func (sm *EditorStateManager) GetStates() []EditorState {
	return sm.states
}

// GetLatestState は最新の状態を返す
func (sm *EditorStateManager) GetLatestState() (EditorState, error) {
	if len(sm.states) == 0 {
		return EditorState{}, fmt.Errorf("no states available")
	}
	return sm.states[len(sm.states)-1], nil
}
