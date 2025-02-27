package events

import (
	"fmt"
	"time"
)

// RecoveryError は復元処理中のエラーを表す
type RecoveryError struct {
	OriginalError error
	SnapshotTime  time.Time
	EventType     EventType
	Context       map[string]interface{}
}

func (e *RecoveryError) Error() string {
	return fmt.Sprintf("recovery failed at %v: %v", e.SnapshotTime, e.OriginalError)
}

// NewRecoveryError は新しいRecoveryErrorを作成する
func NewRecoveryError(err error, timestamp time.Time, eventType EventType) *RecoveryError {
	return &RecoveryError{
		OriginalError: err,
		SnapshotTime:  timestamp,
		EventType:     eventType,
		Context:       make(map[string]interface{}),
	}
}
