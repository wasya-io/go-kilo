package editor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wasya-io/go-kilo/editor/events"
)

// MockEventPublisher はテスト用のイベントパブリッシャー
type MockEventPublisher struct {
	events []Event
}

func NewMockEventPublisher() *MockEventPublisher {
	return &MockEventPublisher{
		events: make([]Event, 0),
	}
}

func (p *MockEventPublisher) BeginBatch() {}
func (p *MockEventPublisher) EndBatch()   {}
func (p *MockEventPublisher) Publish(event Event) error {
	p.events = append(p.events, event)
	return nil
}

// TestStateManagerWithStorage はStateManagerの永続化機能をテストする
func TestStateManagerWithStorage(t *testing.T) {
	// テスト用のディレクトリを作成
	tempDir := filepath.Join(os.TempDir(), "go-kilo-test", time.Now().Format("20060102-150405"))
	defer os.RemoveAll(tempDir)

	storage := NewFileBasedStateStorage(tempDir)
	publisher := NewMockEventPublisher()
	manager := NewEditorStateManager(publisher)
	manager.SetStateStorage(storage)

	t.Run("Create and save snapshots", func(t *testing.T) {
		// 初期状態を作成
		initialEvents := []Event{
			createTestBufferEvent("test1"),
			createTestBufferEvent("test2"),
			createTestBufferEvent("test3"),
		}
		manager.CreateSnapshot(initialEvents)

		// 保存された状態を検証
		states := manager.GetStates()
		if len(states) != 1 {
			t.Fatalf("Expected 1 state, got %d", len(states))
		}
		if len(states[0].Events) != 3 {
			t.Fatalf("Expected 3 events, got %d", len(states[0].Events))
		}

		// さらにイベントを追加
		additionalEvents := []Event{
			createTestBufferEvent("test4"),
			createTestBufferEvent("test5"),
			createTestBufferEvent("test6"),
		}
		manager.CreateSnapshot(append(initialEvents, additionalEvents...))

		// 最新の状態を取得
		latestState, err := manager.GetLatestState()
		if err != nil {
			t.Fatalf("Failed to get latest state: %v", err)
		}
		if len(latestState.Events) != 6 {
			t.Fatalf("Expected 6 events in latest state, got %d", len(latestState.Events))
		}
	})

	t.Run("Load and verify snapshots", func(t *testing.T) {
		// 最新の状態を取得して検証
		latestState, err := manager.GetLatestState()
		if err != nil {
			t.Fatalf("Failed to get latest state: %v", err)
		}
		if len(latestState.Events) != 6 {
			t.Fatalf("Expected 6 events in latest state, got %d", len(latestState.Events))
		}
		for i, event := range latestState.Events {
			t.Logf("Event[%d]: Type=%v", i, event.GetType())
		}
	})

	t.Run("Restore from specific timestamp", func(t *testing.T) {
		states := manager.GetStates()
		firstTimestamp := states[0].Timestamp

		// 特定の時点の状態を復元
		if err := manager.RecoverFromSnapshot(firstTimestamp); err != nil {
			t.Fatalf("Failed to recover from snapshot: %v", err)
		}

		// 復元された状態を検証
		restoredState, err := manager.GetLatestState()
		if err != nil {
			t.Fatalf("Failed to get restored state: %v", err)
		}
		if len(restoredState.Events) != 3 {
			t.Fatalf("Expected 3 events in restored state, got %d", len(restoredState.Events))
		}
	})
}

// TestStateManagerWithMockBuffer はバッファの状態変更と復元をテストする
func TestStateManagerWithMockBuffer(t *testing.T) {
	// EventManagerの初期化
	eventManager := events.NewEventManager()

	// MonitorとRecoveryManagerの初期化
	monitor := events.NewEventMonitor(1000)
	recoveryManager := events.NewRecoveryManager(monitor)
	eventManager.SetRecoveryManager(recoveryManager)

	// StateManagerの初期化
	stateManager := NewEditorStateManager(eventManager)
	recoveryManager.SetStateRecovery(stateManager)

	// バッファの作成
	buffer := NewBuffer(eventManager)

	t.Run("Buffer state changes and recovery", func(t *testing.T) {
		// 1. バッファを初期状態にリセット
		if err := buffer.resetToCleanState(); err != nil {
			t.Fatalf("Failed to reset buffer: %v", err)
		}

		// 2. バッファに文字を挿入
		buffer.InsertChar(events.Position{X: 0, Y: 0}, 'A')
		buffer.InsertChar(events.Position{X: 1, Y: 0}, 'B')
		buffer.InsertChar(events.Position{X: 2, Y: 0}, 'C')

		contentBeforeSnapshot := buffer.GetAllLines()
		t.Logf("Content before snapshot: %v", contentBeforeSnapshot)

		// スナップショット作成前の状態を検証
		if len(contentBeforeSnapshot) != 1 || contentBeforeSnapshot[0] != "ABC" {
			t.Errorf("Initial content incorrect. Expected ['ABC'], got %v", contentBeforeSnapshot)
		}

		firstState := stateManager.CreateSnapshot(eventManager.GetCurrentEvents())

		// スナップショット作成後の状態を検証
		if events := eventManager.GetCurrentEvents(); len(events) == 0 {
			t.Error("No events recorded in event manager after initial changes")
		}

		// 3. 少し待って、タイムスタンプの差を確実にする
		time.Sleep(time.Millisecond * 10) // タイムスタンプの差を確実にするため待機時間を増加

		// 4. 改行を挿入して次の行に 'X' を追加
		buffer.InsertNewline(events.Position{X: 3, Y: 0})
		buffer.InsertChar(events.Position{X: 0, Y: 1}, 'X')

		contentBeforeRestore := buffer.GetAllLines()
		t.Logf("Content before restore: %v", contentBeforeRestore)

		// 復元前の状態を検証
		if len(contentBeforeRestore) != 2 || contentBeforeRestore[0] != "ABC" || contentBeforeRestore[1] != "X" {
			t.Errorf("Content before restore incorrect. Expected ['ABC', 'X'], got %v", contentBeforeRestore)
		}

		secondState := stateManager.CreateSnapshot(eventManager.GetCurrentEvents())

		// スナップショットの検証
		if firstState.Timestamp.Equal(secondState.Timestamp) {
			t.Error("First and second snapshots have the same timestamp")
		}

		// 5. "ABC" の状態に復元（最初のスナップショットの時点）
		if err := stateManager.RecoverFromSnapshot(firstState.Timestamp); err != nil {
			t.Fatalf("Failed to recover from snapshot: %v", err)
		}

		// 6. バッファの状態を検証
		content := buffer.GetAllLines()
		t.Logf("Content after restore: %v", content)

		if len(content) != 1 {
			t.Errorf("Expected 1 line after recovery, got %d lines", len(content))
		}
		if content[0] != "ABC" {
			t.Errorf("Expected content to be 'ABC', got %q", content[0])
		}

		// イベントマネージャーの状態も検証
		restoredEvents := eventManager.GetCurrentEvents()
		if len(restoredEvents) == 0 {
			t.Error("No events present after state restoration")
		}

		// バッファの内部状態の整合性を検証
		if buffer.IsDirty() {
			t.Error("Buffer should not be marked as dirty after restoration")
		}
	})
}

// TestStateManagerStressTest はストレステストを実行する
func TestStateManagerStressTest(t *testing.T) {
	// EventManagerの初期化
	eventManager := events.NewEventManager()

	// StateManagerの初期化
	stateManager := NewEditorStateManager(eventManager)

	// バッファの作成
	buffer := NewBuffer(eventManager)

	t.Run("Rapid state changes and recoveries", func(t *testing.T) {
		// 1. 多数の状態変更を生成
		for i := 0; i < 100; i++ {
			buffer.InsertChar(events.Position{X: i, Y: 0}, rune('A'+(i%26)))
			if i%10 == 0 { // 10文字ごとにスナップショット
				stateManager.CreateSnapshot(eventManager.GetCurrentEvents())
				time.Sleep(time.Millisecond) // わずかな時間差を作る
			}
		}

		// 2. ランダムな時点の状態に復元
		states := stateManager.GetStates()
		if len(states) == 0 {
			t.Fatal("No states available for test")
		}

		// 中間地点のスナップショットを選択
		midPoint := len(states) / 2
		err := stateManager.RecoverFromSnapshot(states[midPoint].Timestamp)
		if err != nil {
			t.Fatalf("Failed to recover from snapshot: %v", err)
		}

		// 3. バッファの状態を検証
		content := buffer.GetAllLines()
		if len(content) == 0 {
			t.Error("Buffer is empty after recovery")
		}

		// 4. 復元後の状態から新しい変更を加える
		for i := 0; i < 50; i++ {
			buffer.InsertChar(events.Position{X: i, Y: 1}, rune('a'+(i%26)))
			if i%10 == 0 {
				stateManager.CreateSnapshot(eventManager.GetCurrentEvents())
			}
		}

		// 最終状態の検証
		finalContent := buffer.GetAllLines()
		if len(finalContent) < 2 {
			t.Error("Expected at least 2 lines after modifications")
		}
	})
}

func TestFileBasedStateStorage(t *testing.T) {
	// テスト用のディレクトリを作成
	tempDir := filepath.Join(os.TempDir(), "go-kilo-test", time.Now().Format("20060102-150405"))
	defer os.RemoveAll(tempDir)

	storage := NewFileBasedStateStorage(tempDir)
	publisher := NewMockEventPublisher()
	manager := NewEditorStateManager(publisher)
	manager.SetStateStorage(storage)

	t.Run("State persistence and recovery", func(t *testing.T) {
		// 初期状態を作成
		initialEvents := []Event{
			createTestBufferEvent("test1"),
			createTestBufferEvent("test2"),
			createTestBufferEvent("test3"),
		}
		manager.CreateSnapshot(initialEvents)

		// 保存された状態を検証
		states := manager.GetStates()
		if len(states) != 1 {
			t.Fatalf("Expected 1 state, got %d", len(states))
		}
		if len(states[0].Events) != 3 {
			t.Fatalf("Expected 3 events, got %d", len(states[0].Events))
		}

		// さらにイベントを追加
		additionalEvents := []Event{
			createTestBufferEvent("test4"),
			createTestBufferEvent("test5"),
			createTestBufferEvent("test6"),
		}
		manager.CreateSnapshot(append(initialEvents, additionalEvents...))

		// 最新の状態を取得
		latestState, err := manager.GetLatestState()
		if err != nil {
			t.Fatalf("Failed to get latest state: %v", err)
		}
		if len(latestState.Events) != 6 {
			t.Fatalf("Expected 6 events in latest state, got %d", len(latestState.Events))
		}

		// 特定の時点の状態を復元
		firstTimestamp := states[0].Timestamp
		if err := manager.RecoverFromSnapshot(firstTimestamp); err != nil {
			t.Fatalf("Failed to recover from snapshot: %v", err)
		}

		// 復元された状態を検証
		restoredState, err := manager.GetLatestState()
		if err != nil {
			t.Fatalf("Failed to get restored state: %v", err)
		}
		if len(restoredState.Events) != 3 {
			t.Fatalf("Expected 3 events in restored state, got %d", len(restoredState.Events))
		}
	})

	t.Run("State cleanup", func(t *testing.T) {
		// クリーンアップ前にファイルが存在することを確認
		if _, err := os.Stat(tempDir); os.IsNotExist(err) {
			t.Fatal("Storage directory does not exist")
		}

		// クリーンアップを実行
		if err := storage.Cleanup(); err != nil {
			t.Fatalf("Failed to cleanup storage: %v", err)
		}

		// クリーンアップ後にファイルが存在しないことを確認
		if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
			t.Fatal("Storage directory still exists after cleanup")
		}
	})
}

// テスト用のバッファイベントを作成するヘルパー関数
func createTestBufferEvent(data string) Event {
	return &BufferEvent{
		BaseEvent: BaseEvent{
			Type:     BufferEventType,
			Time:     time.Now(),
			Priority: 1,
		},
		SubType: events.BufferEventInsert,
		Data:    data,
	}
}
