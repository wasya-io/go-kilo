package core

import (
	"fmt"
	"sync"
	"time"
)

// Metrics はアプリケーションのパフォーマンスメトリクスを収集するインターフェースです。
type Metrics interface {
	RecordEventPublished(eventType string)
	RecordEventHandled(eventType string, handled bool, duration time.Duration)
	RecordRefreshDuration(duration time.Duration)
	RecordEventQueueLength(length int)
	RecordSystemStats(alloc uint64, totalAlloc uint64, sys uint64, numGoroutine int)
	Enabled() bool
}

// MetricsCollector はパフォーマンス情報を集約し、必要に応じてログに出力します。
type MetricsCollector struct {
	enabled bool
	logger  Logger
	mu      sync.Mutex
	// 集計データ
	eventsPublished int64
	eventsHandled   int64
	handledErrors   int64
	lastQueueLen    int
	lastRefreshMs   float64
	lastAlloc       uint64
	lastSys         uint64
	lastNumGoroutine int
}

// NewMetricsCollector は新しい MetricsCollector を作成します。
func NewMetricsCollector(enabled bool, logger Logger) *MetricsCollector {
	return &MetricsCollector{enabled: enabled, logger: logger}
}

func (m *MetricsCollector) Enabled() bool {
	return m != nil && m.enabled
}

func (m *MetricsCollector) RecordEventPublished(eventType string) {
	if !m.Enabled() {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventsPublished++
	if m.logger != nil {
		m.logger.Log("metrics", "event published: "+eventType)
	}
}

func (m *MetricsCollector) RecordEventHandled(eventType string, handled bool, duration time.Duration) {
	if !m.Enabled() {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventsHandled++
	if !handled {
		m.handledErrors++
	}
	if m.logger != nil {
		m.logger.Log("metrics", "event handled: "+eventType+" dur="+duration.String())
	}
}

func (m *MetricsCollector) RecordRefreshDuration(duration time.Duration) {
	if !m.Enabled() {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastRefreshMs = float64(duration.Milliseconds())
	if m.logger != nil {
		m.logger.Log("metrics", "refresh duration: "+duration.String())
	}
}

func (m *MetricsCollector) RecordEventQueueLength(length int) {
	if !m.Enabled() {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastQueueLen = length
	if m.logger != nil {
		m.logger.Log("metrics", "event queue length: "+fmt.Sprintf("%d", length))
	}
}

func (m *MetricsCollector) RecordSystemStats(alloc uint64, totalAlloc uint64, sys uint64, numGoroutine int) {
	if !m.Enabled() {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastAlloc = alloc
	m.lastSys = sys
	m.lastNumGoroutine = numGoroutine
	if m.logger != nil {
		m.logger.Log("metrics", fmt.Sprintf("system stats alloc=%d totalAlloc=%d sys=%d goroutines=%d", alloc, totalAlloc, sys, numGoroutine))
	}
}

func (m *MetricsCollector) Snapshot() map[string]interface{} {
	if !m.Enabled() {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]interface{}{
		"eventsPublished":  m.eventsPublished,
		"eventsHandled":    m.eventsHandled,
		"handledErrors":    m.handledErrors,
		"queueLength":      m.lastQueueLen,
		"refreshMs":        m.lastRefreshMs,
		"allocBytes":       m.lastAlloc,
		"sysBytes":         m.lastSys,
		"goroutines":       m.lastNumGoroutine,
	}
}
