package events

import "time"

// TestState はテスト用の状態を表す構造体
type TestState struct {
	Timestamp time.Time
	Events    []Event
}

// TestStateRecovery はテスト用の状態復元インターフェース
type TestStateRecovery interface {
	StateRecoveryRequester
	GetLastRestoredState() *TestState
	ResetTestState()
}

// TestRecoveryManager はテスト用のリカバリーマネージャー
type TestRecoveryManager struct {
	*RecoveryManager
	failureCount    int
	successCount    int
	lastAttemptedOp BufferOperationType
	lastState       *TestState
}

// NewTestRecoveryManager は新しいテスト用リカバリーマネージャーを作成する
func NewTestRecoveryManager(monitor *EventMonitor) *TestRecoveryManager {
	return &TestRecoveryManager{
		RecoveryManager: NewRecoveryManager(monitor),
	}
}

// AttemptRecovery は状態の復元を試み、統計を記録する
func (trm *TestRecoveryManager) AttemptRecovery(event Event, originalError error) error {
	if bufferEvent, ok := event.(*BufferEvent); ok {
		trm.lastAttemptedOp = bufferEvent.GetOperation()
	}

	err := trm.RecoveryManager.AttemptRecovery(event, originalError)
	if err != nil {
		trm.failureCount++
		return err
	}
	trm.successCount++
	return nil
}

// GetStats はリカバリー試行の統計を返す
func (trm *TestRecoveryManager) GetStats() (successes, failures int) {
	return trm.successCount, trm.failureCount
}

// GetLastAttemptedOperation は最後に試行された操作を返す
func (trm *TestRecoveryManager) GetLastAttemptedOperation() BufferOperationType {
	return trm.lastAttemptedOp
}

// ResetStats は統計をリセットする
func (trm *TestRecoveryManager) ResetStats() {
	trm.successCount = 0
	trm.failureCount = 0
	trm.lastAttemptedOp = ""
}

// MockStateRecovery はテスト用の状態復元実装を作成する
func (trm *TestRecoveryManager) MockStateRecovery(events []Event) *TestState {
	state := &TestState{
		Timestamp: time.Now(),
		Events:    events,
	}
	trm.lastState = state
	return state
}

// GetLastRestoredState は最後に復元された状態を返す
func (trm *TestRecoveryManager) GetLastRestoredState() *TestState {
	return trm.lastState
}

// ResetTestState はテスト状態をリセットする
func (trm *TestRecoveryManager) ResetTestState() {
	trm.lastState = nil
}
