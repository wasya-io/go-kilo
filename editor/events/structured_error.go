package events

import (
	"fmt"
	"runtime/debug"
	"time"
)

// ErrorCategory はエラーの種類を表す
type ErrorCategory int

const (
	// ErrorCategoryUnknown は未分類のエラー
	ErrorCategoryUnknown ErrorCategory = iota
	// ErrorCategoryIO はIO関連のエラー
	ErrorCategoryIO
	// ErrorCategoryState は状態管理関連のエラー
	ErrorCategoryState
	// ErrorCategoryValidation は検証関連のエラー
	ErrorCategoryValidation
	// ErrorCategoryResource はリソース関連のエラー
	ErrorCategoryResource
)

// StructuredError は詳細な情報を持つエラー
type StructuredError struct {
	Category    ErrorCategory
	Message     string
	SourceEvent Event
	Context     map[string]interface{}
	inner       error
}

func (e *StructuredError) Error() string {
	return fmt.Sprintf("[%v] %s: %v", e.Category, e.Message, e.inner)
}

// Unwrap は内部のエラーを返す
func (e *StructuredError) Unwrap() error {
	return e.inner
}

// NewStructuredError は新しいStructuredErrorを作成する
func NewStructuredError(category ErrorCategory, message string, inner error, event Event) *StructuredError {
	return &StructuredError{
		Category:    category,
		Message:     message,
		SourceEvent: event,
		Context:     make(map[string]interface{}),
		inner:       inner,
	}
}

// WithContext はコンテキスト情報を追加する
func (e *StructuredError) WithContext(key string, value interface{}) *StructuredError {
	e.Context[key] = value
	return e
}

// ErrorReport はエラーレポートを生成する
type ErrorReport struct {
	Timestamp    time.Time
	Error        error
	Category     ErrorCategory
	EventType    EventType
	StackTrace   string
	RecoveryInfo *RecoveryInfo
}

// RecoveryInfo はリカバリー試行の情報を保持する
type RecoveryInfo struct {
	AttemptCount   int
	LastStrategy   RecoveryStrategy
	SuccessCount   int
	FailureCount   int
	LastAttemptAt  time.Time
	RecoveredState interface{}
}

// NewErrorReport は新しいErrorReportを作成する
func NewErrorReport(err error, eventType EventType) *ErrorReport {
	category := ErrorCategoryUnknown
	if structured, ok := err.(*StructuredError); ok {
		category = structured.Category
	}

	return &ErrorReport{
		Timestamp:  time.Now(),
		Error:      err,
		Category:   category,
		EventType:  eventType,
		StackTrace: string(debug.Stack()),
	}
}

// WithRecoveryInfo はリカバリー情報を追加する
func (r *ErrorReport) WithRecoveryInfo(info *RecoveryInfo) *ErrorReport {
	r.RecoveryInfo = info
	return r
}
