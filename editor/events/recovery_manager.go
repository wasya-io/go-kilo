package events

import (
	"fmt"
	"time"
)

// RecoveryStrategy は復元戦略を表す
type RecoveryStrategy int

const (
	// LatestSnapshot は最新のスナップショットから復元を試みる
	LatestSnapshot RecoveryStrategy = iota
	// RollbackToStable は最後の安定状態まで戻る
	RollbackToStable
	// IncrementalRecovery は段階的に復元を試みる
	IncrementalRecovery
)

// StateRecoveryRequester は状態の復元を要求するインターフェース
type StateRecoveryRequester interface {
	RequestStateRecovery(timestamp time.Time) error
}

// RecoveryCallback はリカバリー処理のコールバック関数の型
type RecoveryCallback func(event Event, err error) error

// RecoveryManager はリカバリー処理を管理する
type RecoveryManager struct {
	monitor          *EventMonitor
	strategy         RecoveryStrategy
	stateRecovery    StateRecoveryRequester
	recoveryCallback RecoveryCallback
}

// NewRecoveryManager は新しいRecoveryManagerを作成する
func NewRecoveryManager(monitor *EventMonitor) *RecoveryManager {
	return &RecoveryManager{
		monitor:  monitor,
		strategy: LatestSnapshot,
	}
}

// SetStrategy は復元戦略を設定する
func (rm *RecoveryManager) SetStrategy(strategy RecoveryStrategy) {
	rm.strategy = strategy
}

// SetStateRecovery は状態復元の実装を設定する
func (rm *RecoveryManager) SetStateRecovery(recovery StateRecoveryRequester) {
	rm.stateRecovery = recovery
}

// SetRecoveryCallback はリカバリー処理のコールバックを設定する
func (rm *RecoveryManager) SetRecoveryCallback(callback RecoveryCallback) {
	rm.recoveryCallback = callback
}

// AttemptRecovery は状態の復元を試みる
func (rm *RecoveryManager) AttemptRecovery(event Event, originalError error) error {
	if rm.stateRecovery == nil {
		return &RecoveryError{
			OriginalError: fmt.Errorf("no state recovery implementation configured"),
			EventType:     event.GetType(),
		}
	}

	var recoveryTime time.Time
	switch rm.strategy {
	case LatestSnapshot:
		recoveryTime = time.Now()
	case RollbackToStable:
		recoveryTime = time.Now().Add(-5 * time.Minute) // 5分前の状態に戻る
	case IncrementalRecovery:
		recoveryTime = time.Now().Add(-1 * time.Minute) // 1分前の状態に戻る
	}

	if err := rm.stateRecovery.RequestStateRecovery(recoveryTime); err != nil {
		recoveryErr := &RecoveryError{
			OriginalError: err,
			EventType:     event.GetType(),
		}

		// リカバリーコールバックが設定されている場合は呼び出し
		if rm.recoveryCallback != nil {
			if cbErr := rm.recoveryCallback(event, recoveryErr); cbErr != nil {
				// コールバックでもエラーが発生した場合はログに記録
				rm.monitor.LogError(SeverityCritical, event.GetType(), cbErr, "Recovery callback failed")
			}
		}

		return recoveryErr
	}

	rm.monitor.LogError(SeverityInfo, event.GetType(), nil, "Recovery attempt successful")
	return nil
}
