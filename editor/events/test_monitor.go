package events

// TestEventMonitor はテスト用のイベントモニターを提供する
type TestEventMonitor struct {
	*EventMonitor
	lastError     error
	recoveryCount int
	errorCount    map[EventType]int
}

// NewTestEventMonitor は新しいテスト用イベントモニターを作成する
func NewTestEventMonitor(maxLogs int) *TestEventMonitor {
	return &TestEventMonitor{
		EventMonitor: NewEventMonitor(maxLogs),
		errorCount:   make(map[EventType]int),
	}
}

// LogError はエラーを記録し、統計を更新する
func (tm *TestEventMonitor) LogError(severity ErrorSeverity, eventType EventType, err error, msg string) {
	tm.EventMonitor.LogError(severity, eventType, err, msg)
	if severity >= SeverityError {
		tm.errorCount[eventType]++
		tm.lastError = err
	}
}

// GetRecoveryCount は復元試行回数を返す
func (tm *TestEventMonitor) GetRecoveryCount() int {
	return tm.recoveryCount
}

// GetLastError は最後に記録されたエラーを返す
func (tm *TestEventMonitor) GetLastError() error {
	return tm.lastError
}

// GetErrorCountByType はイベントタイプ別のエラー数を返す
func (tm *TestEventMonitor) GetErrorCountByType(eventType EventType) int {
	return tm.errorCount[eventType]
}

// ResetStats は統計をリセットする
func (tm *TestEventMonitor) ResetStats() {
	tm.lastError = nil
	tm.recoveryCount = 0
	tm.errorCount = make(map[EventType]int)
}
