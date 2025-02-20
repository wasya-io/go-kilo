package events

import "time"

// TestRecoveryManager はテスト用のリカバリーマネージャーを提供する
type TestRecoveryManager struct {
	*RecoveryManager
	failureCount    int
	successCount    int
	lastAttemptedOp BufferOperationType
}

// NewTestRecoveryManager は新しいテスト用リカバリーマネージャーを作成する
func NewTestRecoveryManager(em *EventManager, monitor *EventMonitor) *TestRecoveryManager {
	return &TestRecoveryManager{
		RecoveryManager: NewRecoveryManager(em, monitor),
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

// MockSnapshot はテスト用のスナップショットを作成する
func MockSnapshot(timestamp time.Time, events []Event) RecoverySnapshot {
	return RecoverySnapshot{
		Timestamp: timestamp,
		Events:    events,
		States:    make(map[string]interface{}),
	}
}
