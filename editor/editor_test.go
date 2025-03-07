package editor_test

import (
	"os"
	"strings"
	"testing"

	"github.com/wasya-io/go-kilo/app/boundary/logger"
	"github.com/wasya-io/go-kilo/app/boundary/provider/input"
	"github.com/wasya-io/go-kilo/app/boundary/reader"
	"github.com/wasya-io/go-kilo/app/config"
	"github.com/wasya-io/go-kilo/app/entity/contents"
	"github.com/wasya-io/go-kilo/app/entity/key"
	"github.com/wasya-io/go-kilo/app/usecase/parser"
	"github.com/wasya-io/go-kilo/editor"
	"github.com/wasya-io/go-kilo/editor/events"
)

// テスト用のファイルを準備する
func setupTestFile(t *testing.T, content string) (string, func()) {
	t.Helper()
	filename := "test_file.txt"
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("テストファイルの作成に失敗しました: %v", err)
	}
	cleanup := func() {
		os.Remove(filename)
	}
	return filename, cleanup
}

func setupTestEditor(t *testing.T) *editor.Editor {
	t.Helper()
	conf := config.LoadConfig()
	logger := logger.New(conf.DebugMode)

	buffer := contents.NewContents(logger)
	eventManager := events.NewEventManager()
	fileManager := editor.NewFileManager(buffer, eventManager)

	// TODO: モック用のインプットプロバイダを作る
	parser := parser.NewStandardInputParser(logger)
	reader := reader.NewStandardKeyReader(logger)
	inputProvider := input.NewStandardInputProvider(logger, reader, parser)

	ed, err := editor.New(true, conf, logger, eventManager, buffer, fileManager, inputProvider)
	if err != nil {
		t.Fatalf("エディタの初期化に失敗しました: %v", err)
	}
	defer ed.Cleanup()
	return ed
}

// TestEditor はエディタの基本機能をテストする
func TestEditor(t *testing.T) {
	ed := setupTestEditor(t)
	filename, cleanup := setupTestFile(t, "first\nsecond")
	defer cleanup()

	if err := ed.OpenFile(filename); err != nil {
		t.Fatalf("ファイルを開けませんでした: %v", err)
	}

	// 初期状態の確認
	t.Run("初期状態", func(t *testing.T) {
		x, y := ed.TestGetCursor()
		if x != 0 || y != 0 {
			t.Errorf("初期カーソル位置が正しくありません: got (%d,%d), want (0,0)", x, y)
		}

		rows := ed.GetRows()
		if len(rows) != 2 {
			t.Errorf("行数が正しくありません: got %d, want 2", len(rows))
		}
	})

	// カーソル位置の設定
	t.Run("カーソル位置の設定", func(t *testing.T) {
		if err := ed.TestSetCursor(2, 0); err != nil {
			t.Fatalf("カーソル位置の設定に失敗しました: %v", err)
		}

		x, y := ed.TestGetCursor()
		if x != 2 || y != 0 {
			t.Errorf("カーソル位置が正しくありません: got (%d,%d), want (2,0)", x, y)
		}
	})
}

// TestCursorMovement はカーソル移動をテストする
func TestCursorMovement(t *testing.T) {
	ed := setupTestEditor(t)

	filename, cleanup := setupTestFile(t, "first\nsecond")
	defer cleanup()

	if err := ed.OpenFile(filename); err != nil {
		t.Fatalf("ファイルを開けませんでした: %v", err)
	}

	tests := []struct {
		name     string
		dir      byte
		wantX    int
		wantY    int
		initialX int
		initialY int
	}{
		{
			name:     "下に移動",
			dir:      'B',
			initialX: 0,
			initialY: 0,
			wantX:    0,
			wantY:    1,
		},
		{
			name:     "右に移動",
			dir:      'C',
			initialX: 0,
			initialY: 0,
			wantX:    1,
			wantY:    0,
		},
		{
			name:     "上に移動",
			dir:      'A',
			initialX: 0,
			initialY: 1,
			wantX:    0,
			wantY:    0,
		},
		{
			name:     "左に移動",
			dir:      'D',
			initialX: 1,
			initialY: 0,
			wantX:    0,
			wantY:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ed.TestSetCursor(tt.initialX, tt.initialY); err != nil {
				t.Fatalf("カーソル位置の初期化に失敗しました: %v", err)
			}

			if err := ed.TestMoveCursor(editor.CursorMovement(tt.dir)); err != nil {
				t.Fatalf("カーソル移動に失敗しました: %v", err)
			}

			x, y := ed.TestGetCursor()
			if x != tt.wantX || y != tt.wantY {
				t.Errorf("カーソル位置が正しくありません: got (%d,%d), want (%d,%d)",
					x, y, tt.wantX, tt.wantY)
			}
		})
	}
}

// TestCursorMovementWithMultibyte はマルチバイト文字を含む行間のカーソル移動をテストする
func TestCursorMovementWithMultibyte(t *testing.T) {
	ed := setupTestEditor(t)

	// 1行目：ASCII文字のみ（5文字）
	// 2行目：全角文字（3文字）
	// 3行目：ASCII文字のみ（5文字）
	filename, cleanup := setupTestFile(t, "abcde\nあいう\nabcde")
	defer cleanup()

	if err := ed.OpenFile(filename); err != nil {
		t.Fatalf("ファイルを開けませんでした: %v", err)
	}

	tests := []struct {
		name     string
		setup    func(*editor.Editor)
		movement editor.CursorMovement
		wantX    int
		wantY    int
		wantChar string
	}{
		{
			name: "ASCIIの行から全角文字の行へ下移動",
			setup: func(ed *editor.Editor) {
				ed.TestSetCursor(2, 0) // "c"の位置
			},
			movement: editor.CursorDown,
			wantX:    1, // "い"の位置
			wantY:    1,
			wantChar: "い",
		},
		{
			name: "全角文字の行からASCIIの行へ下移動",
			setup: func(ed *editor.Editor) {
				ed.TestSetCursor(1, 1) // "い"の位置
			},
			movement: editor.CursorDown,
			wantX:    2, // "c"の位置
			wantY:    2,
			wantChar: "c",
		},
		{
			name: "ASCIIの行から全角文字の行へ上移動",
			setup: func(ed *editor.Editor) {
				ed.TestSetCursor(2, 2) // 最終行の"c"の位置
			},
			movement: editor.CursorUp,
			wantX:    1, // "い"の位置
			wantY:    1,
			wantChar: "い",
		},
		{
			name: "全角文字の行からASCIIの行へ上移動",
			setup: func(ed *editor.Editor) {
				ed.TestSetCursor(1, 1) // "い"の位置
			},
			movement: editor.CursorUp,
			wantX:    2, // "c"の位置
			wantY:    0,
			wantChar: "c",
		},
		{
			name: "ASCIIの行から全角文字の行へ下移動（行頭位置）",
			setup: func(ed *editor.Editor) {
				ed.TestSetCursor(0, 0) // "a"の位置
			},
			movement: editor.CursorDown,
			wantX:    0, // "あ"の位置
			wantY:    1,
			wantChar: "あ",
		},
		{
			name: "ASCIIの行から全角文字の行へ下移動（中間位置）",
			setup: func(ed *editor.Editor) {
				ed.TestSetCursor(2, 0) // "c"の位置
			},
			movement: editor.CursorDown,
			wantX:    1, // "い"の位置
			wantY:    1,
			wantChar: "い",
		},
		{
			name: "全角文字の行からASCIIの行へ下移動（行頭位置）",
			setup: func(ed *editor.Editor) {
				ed.TestSetCursor(0, 1) // "あ"の位置
			},
			movement: editor.CursorDown,
			wantX:    0, // "a"の位置
			wantY:    2,
			wantChar: "a",
		},
		{
			name: "全角文字の行からASCIIの行へ上移動（中間位置）",
			setup: func(ed *editor.Editor) {
				ed.TestSetCursor(1, 1) // "い"の位置
			},
			movement: editor.CursorUp,
			wantX:    2, // "c"の位置
			wantY:    0,
			wantChar: "c",
		},
		{
			name: "全角文字の行からASCIIの行へ上移動（行頭位置）",
			setup: func(ed *editor.Editor) {
				ed.TestSetCursor(0, 1) // "あ"の位置
			},
			movement: editor.CursorUp,
			wantX:    0, // "a"の位置
			wantY:    0,
			wantChar: "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(ed)

			if err := ed.TestMoveCursor(tt.movement); err != nil {
				t.Fatalf("カーソル移動に失敗しました: %v", err)
			}

			x, y := ed.TestGetCursor()
			if x != tt.wantX || y != tt.wantY {
				t.Errorf("カーソル位置が正しくありません: got (%d,%d), want (%d,%d)",
					x, y, tt.wantX, tt.wantY)
			}

			got := ed.GetCharAtCursor()
			if got != tt.wantChar {
				t.Errorf("カーソル位置の文字が正しくありません: got %q, want %q", got, tt.wantChar)
			}
		})
	}
}

// TestInput は文字入力をテストする
func TestInput(t *testing.T) {
	ed := setupTestEditor(t)

	filename, cleanup := setupTestFile(t, "")
	defer cleanup()

	if err := ed.OpenFile(filename); err != nil {
		t.Fatalf("ファイルを開けませんでした: %v", err)
	}

	tests := []struct {
		name  string
		input rune
		want  string
		wantX int
		wantY int
	}{
		{
			name:  "ASCII文字",
			input: 'a',
			want:  "a",
			wantX: 1,
			wantY: 0,
		},
		{
			name:  "日本語文字",
			input: 'あ',
			want:  "aあ",
			wantX: 2,
			wantY: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ed.TestInput(tt.input); err != nil {
				t.Fatalf("文字の入力に失敗しました: %v", err)
			}

			rows := ed.GetRows()
			if len(rows) == 0 || rows[0] != tt.want {
				t.Errorf("入力結果が正しくありません: got %q, want %q", rows[0], tt.want)
			}

			x, y := ed.TestGetCursor()
			if x != tt.wantX || y != tt.wantY {
				t.Errorf("カーソル位置が正しくありません: got (%d,%d), want (%d,%d)",
					x, y, tt.wantX, tt.wantY)
			}
		})
	}
}

// TestDelete はバックスペースをテストする
func TestDelete(t *testing.T) {
	ed := setupTestEditor(t)

	filename, cleanup := setupTestFile(t, "abc")
	defer cleanup()

	if err := ed.OpenFile(filename); err != nil {
		t.Fatalf("ファイルを開けませんでした: %v", err)
	}

	if err := ed.TestSetCursor(3, 0); err != nil {
		t.Fatalf("カーソル位置の設定に失敗しました: %v", err)
	}

	// バックスペースを実行
	if err := ed.TestDelete(); err != nil {
		t.Fatalf("バックスペースの処理に失敗しました: %v", err)
	}

	// 結果を検証
	rows := ed.GetRows()
	if len(rows) == 0 || rows[0] != "ab" {
		t.Errorf("削除結果が正しくありません: got %q, want %q", rows[0], "ab")
	}

	x, y := ed.TestGetCursor()
	if x != 2 || y != 0 {
		t.Errorf("カーソル位置が正しくありません: got (%d,%d), want (2,0)", x, y)
	}
}

func TestEditorKeyInput(t *testing.T) {
	// テストケース用のキーイベントを準備
	events := []key.KeyEvent{
		{Type: key.KeyEventChar, Rune: 'H'},
		{Type: key.KeyEventChar, Rune: 'i'},
		{Type: key.KeyEventSpecial, Key: key.KeyEnter},
		{Type: key.KeyEventChar, Rune: '!'},
	}

	// MockKeyReaderを使用してエディタを初期化
	e := setupTestEditor(t)

	// TODO: mockReaderを注入すればSetKeyReaderが不要になる
	// mockReader := editor.NewMockKeyReader(events)
	// e.SetKeyReader(mockReader)

	// キー入力をシミュレート
	for i := 0; i < len(events); i++ {
		err := e.ProcessKeypress()
		if err != nil {
			t.Errorf("Failed to process keypress: %v", err)
		}
	}

	// 期待される結果を検証
	expectedLines := []string{
		"Hi",
		"!",
	}

	lineCount := e.GetLineCount()
	if lineCount != len(expectedLines) {
		t.Errorf("Expected %d lines, got %d", len(expectedLines), lineCount)
	}

	for i, expected := range expectedLines {
		if i >= lineCount {
			break
		}
		if content := e.GetContent(i); content != expected {
			t.Errorf("Line %d: expected '%s', got '%s'", i, expected, content)
		}
	}
}

// TestGetCharAtCursor はカーソル位置の文字を取得する機能をテストする
func TestGetCharAtCursor(t *testing.T) {
	ed := setupTestEditor(t)

	filename, cleanup := setupTestFile(t, "abc\nあいう")
	defer cleanup()

	if err := ed.OpenFile(filename); err != nil {
		t.Fatalf("ファイルを開けませんでした: %v", err)
	}

	tests := []struct {
		name     string
		x        int
		y        int
		expected string
	}{
		{
			name:     "ASCII文字",
			x:        0,
			y:        0,
			expected: "a",
		},
		{
			name:     "日本語文字",
			x:        0,
			y:        1,
			expected: "あ",
		},
		{
			name:     "行末",
			x:        2,
			y:        0,
			expected: "c",
		},
		{
			name:     "存在しない位置",
			x:        10,
			y:        0,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ed.TestSetCursor(tt.x, tt.y); err != nil {
				t.Fatalf("カーソル位置の設定に失敗しました: %v", err)
			}

			got := ed.GetCharAtCursor()
			if got != tt.expected {
				t.Errorf("GetCharAtCursor() = %q, want %q", got, tt.expected)
			}
		})
	}

	// 存在しない行のテストは別途実行
	t.Run("存在しない行", func(t *testing.T) {
		// 空のファイルを作成してテスト
		emptyFile, cleanup := setupTestFile(t, "")
		defer cleanup()

		if err := ed.OpenFile(emptyFile); err != nil {
			t.Fatalf("空ファイルを開けませんでした: %v", err)
		}

		got := ed.GetCharAtCursor()
		if got != "" {
			t.Errorf("GetCharAtCursor() = %q, want %q", got, "")
		}
	})
}

// TestEscapeSequence はESCキーとエスケープシーケンスの処理をテストする
func TestEscapeSequence(t *testing.T) {
	ed := setupTestEditor(t)

	tests := []struct {
		name     string
		input    []key.KeyEvent
		wantFunc func(*testing.T, *editor.Editor)
	}{
		{
			name: "ESCキー単体",
			input: []key.KeyEvent{
				{Type: key.KeyEventSpecial, Key: key.KeyEsc},
			},
			wantFunc: func(t *testing.T, e *editor.Editor) {
				// ESCキーは特に状態を変更しないので、
				// 正常に処理されたことだけを確認
				if e.IsDirty() {
					t.Error("バッファが変更されてはいけません")
				}
			},
		},
		{
			name: "矢印キー（エスケープシーケンス）",
			input: []key.KeyEvent{
				{Type: key.KeyEventSpecial, Key: key.KeyArrowRight},
				{Type: key.KeyEventSpecial, Key: key.KeyArrowLeft},
			},
			wantFunc: func(t *testing.T, e *editor.Editor) {
				// カーソルが元の位置に戻っていることを確認
				x, y := e.TestGetCursor()
				if x != 0 || y != 0 {
					t.Errorf("カーソル位置が正しくありません: got (%d,%d), want (0,0)", x, y)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用のキーリーダーを設定
			// mockReader := editor.NewMockKeyReader(tt.input)
			// ed.SetKeyReader(mockReader)

			// 各キー入力を処理
			for range tt.input {
				if err := ed.ProcessKeypress(); err != nil {
					t.Fatalf("キー処理でエラーが発生: %v", err)
				}
			}

			// 結果を検証
			tt.wantFunc(t, ed)
		})
	}
}

func TestTabHandling(t *testing.T) {
	// Create a temporary .env file with custom TAB_WIDTH
	os.Setenv("TAB_WIDTH", "4")

	ed := setupTestEditor(t)

	filename, cleanup := setupTestFile(t, "")
	defer cleanup()

	if err := ed.OpenFile(filename); err != nil {
		t.Fatalf("ファイルを開けませんでした: %v", err)
	}

	// Create test events
	events := []key.KeyEvent{
		{Type: key.KeyEventSpecial, Key: key.KeyTab},
		{Type: key.KeyEventChar, Rune: 'a'},
	}

	// Set up mock key reader
	// mockReader := editor.NewMockKeyReader(events)
	// ed.SetKeyReader(mockReader)

	// Process key events
	for range events {
		if err := ed.ProcessKeypress(); err != nil {
			t.Fatalf("キー処理でエラーが発生: %v", err)
		}
	}

	// Verify the content
	content := ed.GetContent(0)
	expectedSpaces := strings.Repeat(" ", 4) // TAB_WIDTH=4 from env
	expected := expectedSpaces + "a"

	if content != expected {
		t.Errorf("タブの展開が正しくありません: got %q, want %q", content, expected)
	}
}

func TestShiftTabHandling(t *testing.T) {
	// Create test cases with different TAB_WIDTH settings
	tests := []struct {
		name      string
		tabWidth  string
		content   string
		cursorX   int
		wantCount int
	}{
		{
			name:      "TAB_WIDTH=4: delete 4 spaces",
			tabWidth:  "4",
			content:   "    abc",
			cursorX:   4,
			wantCount: 0,
		},
		{
			name:      "TAB_WIDTH=4: delete 1 space from 5 spaces",
			tabWidth:  "4",
			content:   "     abc",
			cursorX:   5,
			wantCount: 4,
		},
		{
			name:      "TAB_WIDTH=4: delete 2 spaces from 6 spaces",
			tabWidth:  "4",
			content:   "      abc",
			cursorX:   6,
			wantCount: 4,
		},
		{
			name:      "TAB_WIDTH=2: delete 2 spaces",
			tabWidth:  "2",
			content:   "  abc",
			cursorX:   2,
			wantCount: 0,
		},
		{
			name:      "TAB_WIDTH=2: delete 1 space from 3 spaces",
			tabWidth:  "2",
			content:   "   abc",
			cursorX:   3,
			wantCount: 2,
		},
		{
			name:      "TAB_WIDTH=2: delete 1 space from 3 spaces",
			tabWidth:  "2",
			content:   "  a  bc",
			cursorX:   5,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set TAB_WIDTH environment variable
			os.Setenv("TAB_WIDTH", tt.tabWidth)
			defer os.Unsetenv("TAB_WIDTH")

			ed := setupTestEditor(t)

			// Set initial content and cursor position
			filename, cleanup := setupTestFile(t, tt.content)
			defer cleanup()

			if err := ed.OpenFile(filename); err != nil {
				t.Fatalf("ファイルを開けませんでした: %v", err)
			}

			if err := ed.TestSetCursor(tt.cursorX, 0); err != nil {
				t.Fatalf("カーソル位置の設定に失敗しました: %v", err)
			}

			// Create mock key event for Shift+Tab
			// events := []editor.KeyEvent{
			// 	{Type: editor.KeyEventSpecial, Key: editor.KeyShiftTab},
			// }
			// mockReader := editor.NewMockKeyReader(events)
			// ed.SetKeyReader(mockReader)

			// Process Shift+Tab key event
			if err := ed.ProcessKeypress(); err != nil {
				t.Fatalf("キー処理でエラーが発生: %v", err)
			}

			// Verify the result
			content := ed.GetContent(0)
			leadingSpaces := countLeadingSpaces(content)
			if leadingSpaces != tt.wantCount {
				t.Errorf("先頭スペース数が正しくありません: got %d, want %d", leadingSpaces, tt.wantCount)
			}
		})
	}
}

// countLeadingSpaces は文字列の先頭の空白文字数を数える
func countLeadingSpaces(s string) int {
	count := 0
	for _, r := range s {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}
