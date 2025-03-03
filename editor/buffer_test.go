package editor

import (
	"testing"

	"github.com/wasya-io/go-kilo/editor/events"
)

func TestBufferBasicOperations(t *testing.T) {
	eventManager := events.NewEventManager()

	t.Run("InsertChar position accuracy", func(t *testing.T) {
		buffer := NewBuffer(eventManager)

		// 1. 空のバッファへの挿入
		buffer.InsertChar(events.Position{X: 0, Y: 0}, 'A')
		content := buffer.GetAllLines()
		if len(content) != 1 || content[0] != "A" {
			t.Errorf("Expected ['A'], got %v", content)
		}

		// 2. 行の途中への挿入
		buffer.InsertChar(events.Position{X: 1, Y: 0}, 'B')
		buffer.InsertChar(events.Position{X: 1, Y: 0}, 'C') // ABの間にCを挿入
		content = buffer.GetAllLines()
		if content[0] != "ACB" {
			t.Errorf("Expected 'ACB', got %s", content[0])
		}

		// 3. 不正な位置への挿入（範囲外）
		buffer.InsertChar(events.Position{X: 10, Y: 10}, 'X')
		content = buffer.GetAllLines()
		if content[0] != "ACB" { // 変更されていないことを確認
			t.Errorf("Buffer changed with invalid position insert. Got %s", content[0])
		}
	})

	t.Run("DeleteChar and line joining", func(t *testing.T) {
		buffer := NewBuffer(eventManager)

		// 1. 基本的な削除（バックスペースの挙動）
		buffer.InsertChar(events.Position{X: 0, Y: 0}, 'A')
		buffer.InsertChar(events.Position{X: 1, Y: 0}, 'B')
		buffer.DeleteChar(events.Position{X: 1, Y: 0}) // カーソルがBの位置にある時のバックスペース

		content := buffer.GetAllLines()
		if content[0] != "B" { // 'A'が削除され、'B'が残る
			t.Errorf("Expected 'B', got %s", content[0])
		}

		// 2. 行の結合（バックスペースによる）
		buffer.InsertNewline(events.Position{X: 1, Y: 0})
		buffer.InsertChar(events.Position{X: 0, Y: 1}, 'C')
		buffer.DeleteChar(events.Position{X: 0, Y: 1}) // 行頭でのバックスペース

		content = buffer.GetAllLines()
		if len(content) != 1 || content[0] != "BC" {
			t.Errorf("Expected ['BC'], got %v", content)
		}
	})

	t.Run("InsertNewline splits and joins", func(t *testing.T) {
		buffer := NewBuffer(eventManager)

		// 1. 基本的な行分割
		buffer.InsertChar(events.Position{X: 0, Y: 0}, 'A')
		buffer.InsertChar(events.Position{X: 1, Y: 0}, 'B')
		buffer.InsertNewline(events.Position{X: 1, Y: 0})

		content := buffer.GetAllLines()
		if len(content) != 2 || content[0] != "A" || content[1] != "B" {
			t.Errorf("Expected ['A', 'B'], got %v", content)
		}

		// 2. 空行の挿入
		buffer.InsertNewline(events.Position{X: 0, Y: 1})
		content = buffer.GetAllLines()
		if len(content) != 3 {
			t.Errorf("Expected 3 lines, got %d", len(content))
		}
	})

	t.Run("GetAllLines state consistency", func(t *testing.T) {
		buffer := NewBuffer(eventManager)

		// 1. 一連の操作後の状態一貫性
		operations := []struct {
			pos events.Position
			ch  rune
		}{
			{events.Position{X: 0, Y: 0}, 'H'},
			{events.Position{X: 1, Y: 0}, 'e'},
			{events.Position{X: 2, Y: 0}, 'l'},
			{events.Position{X: 3, Y: 0}, 'l'},
			{events.Position{X: 4, Y: 0}, 'o'},
		}

		for _, op := range operations {
			buffer.InsertChar(op.pos, op.ch)
		}

		content := buffer.GetAllLines()
		if len(content) != 1 || content[0] != "Hello" {
			t.Errorf("Expected ['Hello'], got %v", content)
		}

		// 2. 削除操作後の一貫性（バックスペースでの削除）
		buffer.DeleteChar(events.Position{X: 5, Y: 0}) // カーソルが末尾にある時のバックスペース
		buffer.DeleteChar(events.Position{X: 4, Y: 0}) // カーソルが'o'の位置にある時のバックスペース

		content = buffer.GetAllLines()
		if len(content) != 1 || content[0] != "Hel" {
			t.Errorf("Expected ['Hel'], got %v", content)
		}
	})

	t.Run("Cursor boundary check", func(t *testing.T) {
		buffer := NewBuffer(eventManager)

		// 1. 空バッファでの境界チェック
		if buffer.GetLineCount() != 0 {
			t.Errorf("Empty buffer should have 0 lines, got %d", buffer.GetLineCount())
		}

		// 2. 単一行での境界チェック
		buffer.InsertChar(events.Position{X: 0, Y: 0}, 'A')
		row := buffer.getRow(0)
		if row == nil {
			t.Error("Expected non-nil row at position 0")
		} else {
			if row.GetRuneCount() != 1 {
				t.Errorf("Expected rune count 1, got %d", row.GetRuneCount())
			}
		}

		// 3. 不正な位置のアクセス
		row = buffer.getRow(-1)
		if row != nil {
			t.Error("Expected nil row for negative index")
		}

		row = buffer.getRow(9999)
		if row != nil {
			t.Error("Expected nil row for out of bounds index")
		}
	})
}

func TestBufferStateManagement(t *testing.T) {
	eventManager := events.NewEventManager()

	t.Run("Buffer state consistency", func(t *testing.T) {
		buffer := NewBuffer(eventManager)

		// 1. 初期状態の確認
		if buffer.IsDirty() {
			t.Error("New buffer should not be dirty")
		}

		// 2. 変更による状態変化
		buffer.InsertChar(events.Position{X: 0, Y: 0}, 'A')
		if !buffer.IsDirty() {
			t.Error("Buffer should be dirty after modification")
		}

		// 3. 複数行の状態
		buffer.InsertNewline(events.Position{X: 1, Y: 0})
		buffer.InsertChar(events.Position{X: 0, Y: 1}, 'B')

		state := buffer.getCurrentState()
		if len(state.Lines) != 2 {
			t.Errorf("Expected 2 lines in state, got %d", len(state.Lines))
		}
		if state.Lines[0] != "A" || state.Lines[1] != "B" {
			t.Errorf("Expected ['A', 'B'], got %v", state.Lines)
		}
	})

	t.Run("State restoration", func(t *testing.T) {
		buffer := NewBuffer(eventManager)

		// 1. 初期状態の保存
		buffer.InsertChar(events.Position{X: 0, Y: 0}, 'A')
		buffer.InsertChar(events.Position{X: 1, Y: 0}, 'B')
		initialState := buffer.getCurrentState()

		// 2. 追加の変更
		buffer.InsertNewline(events.Position{X: 2, Y: 0})
		buffer.InsertChar(events.Position{X: 0, Y: 1}, 'C')

		// 3. 状態を復元
		err := buffer.RestoreState(initialState)
		if err != nil {
			t.Errorf("Failed to restore state: %v", err)
		}

		content := buffer.GetAllLines()
		if len(content) != 1 || content[0] != "AB" {
			t.Errorf("Expected ['AB'] after restore, got %v", content)
		}
	})

	t.Run("Clean state reset", func(t *testing.T) {
		buffer := NewBuffer(eventManager)

		// 1. バッファに内容を追加
		buffer.InsertChar(events.Position{X: 0, Y: 0}, 'A')
		buffer.InsertNewline(events.Position{X: 1, Y: 0})
		buffer.InsertChar(events.Position{X: 0, Y: 1}, 'B')

		// 2. クリーン状態にリセット
		err := buffer.resetToCleanState()
		if err != nil {
			t.Errorf("Failed to reset to clean state: %v", err)
		}

		// 3. リセット後の状態を検証
		if buffer.IsDirty() {
			t.Error("Buffer should not be dirty after reset")
		}
		content := buffer.GetAllLines()
		if len(content) != 1 || content[0] != "" {
			t.Errorf("Expected [''], got %v", content)
		}
	})

	t.Run("Row content consistency", func(t *testing.T) {
		buffer := NewBuffer(eventManager)

		// 1. 行の追加と内容の確認
		buffer.InsertChar(events.Position{X: 0, Y: 0}, 'A')
		buffer.InsertChar(events.Position{X: 1, Y: 0}, 'B')

		row := buffer.getRow(0)
		if row == nil {
			t.Fatal("Expected non-nil row")
		}

		// 2. Row.GetContent()の一貫性チェック
		content := row.GetContent()
		if content != "AB" {
			t.Errorf("Expected 'AB', got %q", content)
		}

		// 3. 文字の削除後の一貫性
		buffer.DeleteChar(events.Position{X: 2, Y: 0}) // ABの後ろでバックスペース
		content = row.GetContent()
		if content != "A" {
			t.Errorf("Expected 'A' after delete, got %q", content)
		}
	})
}

func TestBufferStateOperations(t *testing.T) {
	t.Run("Compare buffer states", func(t *testing.T) {
		tests := []struct {
			name     string
			stateA   events.BufferState
			stateB   events.BufferState
			expected bool
		}{
			{
				name: "identical states",
				stateA: events.BufferState{
					Content: "test",
					IsDirty: true,
					Lines:   []string{"test"},
				},
				stateB: events.BufferState{
					Content: "test",
					IsDirty: true,
					Lines:   []string{"test"},
				},
				expected: true,
			},
			{
				name: "different content",
				stateA: events.BufferState{
					Content: "test1",
					IsDirty: true,
					Lines:   []string{"test1"},
				},
				stateB: events.BufferState{
					Content: "test2",
					IsDirty: true,
					Lines:   []string{"test2"},
				},
				expected: false,
			},
			{
				name: "different dirty flag",
				stateA: events.BufferState{
					Content: "test",
					IsDirty: true,
					Lines:   []string{"test"},
				},
				stateB: events.BufferState{
					Content: "test",
					IsDirty: false,
					Lines:   []string{"test"},
				},
				expected: false,
			},
			{
				name: "different line count",
				stateA: events.BufferState{
					Content: "test",
					IsDirty: true,
					Lines:   []string{"test"},
				},
				stateB: events.BufferState{
					Content: "test",
					IsDirty: true,
					Lines:   []string{"test", ""},
				},
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := compareBufferStates(tt.stateA, tt.stateB)
				if result != tt.expected {
					t.Errorf("compareBufferStates() = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("Get current state accuracy", func(t *testing.T) {
		eventManager := events.NewEventManager()
		buffer := NewBuffer(eventManager)

		// 1. 空の状態
		state := buffer.getCurrentState()
		if state.IsDirty {
			t.Error("Empty buffer should not be dirty")
		}
		if len(state.Lines) != 0 {
			t.Errorf("Empty buffer should have no lines, got %d", len(state.Lines))
		}

		// 2. 単一行の状態
		buffer.InsertChar(events.Position{X: 0, Y: 0}, 'A')
		state = buffer.getCurrentState()
		if !state.IsDirty {
			t.Error("Buffer should be dirty after modification")
		}
		if state.Content != "A" {
			t.Errorf("Expected content 'A', got %q", state.Content)
		}
		if len(state.Lines) != 1 || state.Lines[0] != "A" {
			t.Errorf("Expected lines ['A'], got %v", state.Lines)
		}

		// 3. 複数行の状態
		buffer.InsertNewline(events.Position{X: 1, Y: 0})
		buffer.InsertChar(events.Position{X: 0, Y: 1}, 'B')
		state = buffer.getCurrentState()
		if len(state.Lines) != 2 {
			t.Errorf("Expected 2 lines, got %d", len(state.Lines))
		}
		if state.Lines[0] != "A" || state.Lines[1] != "B" {
			t.Errorf("Expected lines ['A', 'B'], got %v", state.Lines)
		}
		if state.Content != "A" { // Content should be the first line
			t.Errorf("Expected content 'A', got %q", state.Content)
		}
	})
}

func TestBufferEventPublishing(t *testing.T) {
	t.Run("Event publishing behavior", func(t *testing.T) {
		// モックイベントマネージャーを作成
		mockManager := events.NewEventManager()
		var lastEvent events.Event
		mockManager.Subscribe(events.BufferEventType, func(event events.Event) {
			lastEvent = event
		})

		buffer := NewBuffer(mockManager)

		// 1. 通常の変更イベント
		buffer.InsertChar(events.Position{X: 0, Y: 0}, 'A')
		if lastEvent == nil {
			t.Fatal("No event published for InsertChar")
		}
		if bufferEvent, ok := lastEvent.(*events.BufferEvent); ok {
			if bufferEvent.SubType != events.BufferContentChanged {
				t.Errorf("Expected BufferContentChanged event, got %v", bufferEvent.SubType)
			}
			prevState, currState := bufferEvent.GetStates()
			if len(prevState.Lines) != 0 {
				t.Errorf("Previous state should be empty, got %v", prevState.Lines)
			}
			if len(currState.Lines) != 1 || currState.Lines[0] != "A" {
				t.Errorf("Current state should be ['A'], got %v", currState.Lines)
			}
		}

		// 2. 構造的変更イベント
		lastEvent = nil
		buffer.InsertNewline(events.Position{X: 1, Y: 0})
		if lastEvent == nil {
			t.Fatal("No event published for InsertNewline")
		}
		if bufferEvent, ok := lastEvent.(*events.BufferEvent); ok {
			if bufferEvent.SubType != events.BufferStructuralChange {
				t.Errorf("Expected BufferStructuralChange event, got %v", bufferEvent.SubType)
			}
			prevState, currState := bufferEvent.GetStates()
			if len(prevState.Lines) != 1 || prevState.Lines[0] != "A" {
				t.Errorf("Previous state should be ['A'], got %v", prevState.Lines)
			}
			if len(currState.Lines) != 2 || currState.Lines[0] != "A" || currState.Lines[1] != "" {
				t.Errorf("Current state should be ['A', ''], got %v", currState.Lines)
			}
		}

		// 3. 状態設定イベント
		lastEvent = nil
		initialState := events.BufferState{
			Content: "test",
			IsDirty: false,
			Lines:   []string{"test"},
		}
		buffer.publishBufferEvent(events.BufferEventSetState, events.Position{}, initialState, events.BufferState{})
		if lastEvent == nil {
			t.Fatal("No event published for state setting")
		}
		if bufferEvent, ok := lastEvent.(*events.BufferEvent); ok {
			if bufferEvent.SubType != events.BufferEventSetState {
				t.Errorf("Expected BufferEventSetState event, got %v", bufferEvent.SubType)
			}
			prevState, currState := bufferEvent.GetStates()
			// 状態設定イベントでは、前の状態は空であるべき
			if len(prevState.Lines) != 1 || prevState.Lines[0] != "" {
				t.Errorf("Previous state should be [''], got %v", prevState.Lines)
			}
			// 現在の状態は設定された状態を反映しているべき
			if len(currState.Lines) != 1 || currState.Lines[0] != "" {
				t.Errorf("Current state should be [''], got %v", currState.Lines)
			}
		}
	})

	t.Run("Event batch processing", func(t *testing.T) {
		mockManager := events.NewEventManager()
		batchCount := 0
		mockManager.Subscribe(events.BufferEventType, func(event events.Event) {
			if bufferEvent, ok := event.(*events.BufferEvent); ok {
				if bufferEvent.SubType == events.BufferContentChanged {
					batchCount++
				}
			}
		})

		buffer := NewBuffer(mockManager)

		// 複数の文字を一度に挿入（バッチ処理されるべき）
		chars := []rune("Hello")
		buffer.InsertChars(events.Position{X: 0, Y: 0}, chars)

		// バッチ処理により、1つのイベントのみが発行されるべき
		if batchCount != 1 {
			t.Errorf("Expected 1 batch event, got %d events", batchCount)
		}
	})
}
