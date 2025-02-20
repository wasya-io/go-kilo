package events

import (
	"fmt"
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

// RecoveryManager はリカバリー処理を管理する
type RecoveryManager struct {
	eventManager *EventManager
	monitor      *EventMonitor
	strategy     RecoveryStrategy
}

// NewRecoveryManager は新しいRecoveryManagerを作成する
func NewRecoveryManager(em *EventManager, monitor *EventMonitor) *RecoveryManager {
	return &RecoveryManager{
		eventManager: em,
		monitor:      monitor,
		strategy:     LatestSnapshot,
	}
}

// SetStrategy は復元戦略を設定する
func (rm *RecoveryManager) SetStrategy(strategy RecoveryStrategy) {
	rm.strategy = strategy
}

// AttemptRecovery は状態の復元を試みる
func (rm *RecoveryManager) AttemptRecovery(event Event, originalError error) error {
	switch rm.strategy {
	case LatestSnapshot:
		return rm.recoverFromLatestSnapshot(event)
	case RollbackToStable:
		return rm.rollbackToStable(event)
	case IncrementalRecovery:
		return rm.incrementalRecovery(event)
	default:
		return fmt.Errorf("unknown recovery strategy")
	}
}

// recoverFromLatestSnapshot は最新のスナップショットから復元を試みる
func (rm *RecoveryManager) recoverFromLatestSnapshot(event Event) error {
	snapshots := rm.eventManager.snapshots
	if len(snapshots) == 0 {
		return &RecoveryError{
			OriginalError: fmt.Errorf("no snapshots available"),
			EventType:     event.GetType(),
		}
	}

	latestSnapshot := snapshots[len(snapshots)-1]
	if err := rm.eventManager.RecoverFromSnapshot(latestSnapshot.Timestamp); err != nil {
		return &RecoveryError{
			OriginalError: err,
			SnapshotTime:  latestSnapshot.Timestamp,
			EventType:     event.GetType(),
		}
	}

	rm.monitor.LogError(SeverityInfo, event.GetType(), nil, "Successfully recovered from latest snapshot")
	return nil
}

// rollbackToStable は最後の安定状態まで戻る
func (rm *RecoveryManager) rollbackToStable(event Event) error {
	// 最新から順に安定状態を探す
	for i := len(rm.eventManager.snapshots) - 1; i >= 0; i-- {
		snapshot := rm.eventManager.snapshots[i]
		if err := rm.eventManager.RecoverFromSnapshot(snapshot.Timestamp); err == nil {
			rm.monitor.LogError(SeverityInfo, event.GetType(), nil, "Rolled back to stable state")
			return nil
		}
	}

	return &RecoveryError{
		OriginalError: fmt.Errorf("no stable state found"),
		EventType:     event.GetType(),
	}
}

// incrementalRecovery は段階的に復元を試みる
func (rm *RecoveryManager) incrementalRecovery(event Event) error {
	// 最新のスナップショットから1つずつ適用
	var lastError error
	for _, snapshot := range rm.eventManager.snapshots {
		if err := rm.eventManager.RecoverFromSnapshot(snapshot.Timestamp); err != nil {
			lastError = err
			continue
		}
		rm.monitor.LogError(SeverityInfo, event.GetType(), nil, "Incremental recovery succeeded")
		return nil
	}

	return &RecoveryError{
		OriginalError: lastError,
		EventType:     event.GetType(),
	}
}
