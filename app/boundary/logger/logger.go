package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// LogEntry はログのエントリを表す構造体
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Type      string `json:"type"`
}

// Logger はロギング機能を提供する構造体
type Logger struct {
	debugMode bool
	entries   []LogEntry
	filePath  string
	maxBuffer int
	startTime time.Time
}

// New は新しいLoggerインスタンスを作成する
func New(debugMode bool) *Logger {
	startTime := time.Now()
	return &Logger{
		debugMode: debugMode,
		entries:   make([]LogEntry, 0),
		filePath:  fmt.Sprintf("log-%s.json", startTime.Format("20060102-150405")),
		maxBuffer: 100,
		startTime: startTime,
	}
}

// Log はメッセージをログに記録する
func (l *Logger) Log(messageType string, message string) {
	if !l.debugMode {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   message,
		Type:      messageType,
	}
	l.entries = append(l.entries, entry)

	// バッファが一定量に達したらフラッシュ
	if len(l.entries) >= l.maxBuffer {
		l.Flush()
	}
}

// Flush は現在のログエントリをファイルに書き出す
func (l *Logger) Flush() {
	if len(l.entries) == 0 {
		return
	}

	// ログをJSONとして書き出す
	data, err := json.MarshalIndent(l.entries, "", "  ")
	if err == nil {
		os.WriteFile(l.filePath, data, 0644)
	}

	// ログをクリア
	l.entries = []LogEntry{}
}

// SetDebugMode はデバッグモードの状態を設定する
func (l *Logger) SetDebugMode(enabled bool) {
	l.debugMode = enabled
}
