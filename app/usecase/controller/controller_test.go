package controller

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	mock_filemanager "github.com/wasya-io/go-kilo/app/boundary/filemanager/mock"
	mock_input "github.com/wasya-io/go-kilo/app/boundary/provider/input/mock"
	mock_writer "github.com/wasya-io/go-kilo/app/boundary/writer/mock"
	"github.com/wasya-io/go-kilo/app/entity/contents"
	mock_contents "github.com/wasya-io/go-kilo/app/entity/contents/mock"
	mock_core "github.com/wasya-io/go-kilo/app/entity/core/mock"
	"github.com/wasya-io/go-kilo/app/entity/cursor"
	"github.com/wasya-io/go-kilo/app/entity/event"
	"github.com/wasya-io/go-kilo/app/entity/key"
	"github.com/wasya-io/go-kilo/app/entity/screen"
)

func TestController_Process(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFileManager := mock_filemanager.NewMockFileManager(ctrl)
	mockInputProvider := mock_input.NewMockProvider(ctrl)
	mockLogger := mock_core.NewMockLogger(ctrl)
	mockWriter := mock_writer.NewMockScreenWriter(ctrl)
	mockBuilder := mock_contents.NewMockBuilder(ctrl)
	mockEventBus := event.NewBus()

	// 基本的な依存関係のセットアップ
	c := contents.NewContents(mockLogger)
	cur := cursor.NewCursor()
	msg := contents.NewMessage("")
	scr := screen.NewScreen(mockBuilder, mockWriter, msg, cur, 24, 80)

	controller := NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, mockEventBus)

	// Process のモック期待値
	// 1. readEvent は GetInputEvents を呼び出す
	mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventChar, Rune: 'a'}, nil, nil)

	// 2. createCommand はコマンドを作成する（内部ロジックだが、効果やログを検証できる）
	// 'a' の場合、InsertCharCommand を作成する。
	// リファクタリングなしに内部の createCommand の戻り値を簡単にモックすることはできないが、
	// コマンド実行の副作用を検証することはできる。

	// RefreshScreen の呼び出しを期待する
	// RefreshScreen の呼び出し内容:
	// - fileManager.GetFilename()
	// - logger.Log("screen", ...)
	// - screen.Redraw -> builder calls, writer calls
	mockFileManager.EXPECT().GetFilename().Return("test.txt").AnyTimes()
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()

	// 画面再描画の期待値（簡略化）
	mockBuilder.EXPECT().Clear().AnyTimes()
	mockBuilder.EXPECT().Write(gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Build().Return("mock_screen_content").AnyTimes()
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()

	// Process を実行する
	err := controller.Process()
	assert.NoError(t, err)

	// 'a' が挿入されたことを検証する
	lines := c.GetAllLines()
	assert.Equal(t, 1, len(lines))
	assert.Equal(t, "a", lines[0])
}

func TestController_EditorOperations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFileManager := mock_filemanager.NewMockFileManager(ctrl)
	mockInputProvider := mock_input.NewMockProvider(ctrl)
	mockLogger := mock_core.NewMockLogger(ctrl)
	mockWriter := mock_writer.NewMockScreenWriter(ctrl)
	mockBuilder := mock_contents.NewMockBuilder(ctrl)
	mockEventBus := event.NewBus()

	c := contents.NewContents(mockLogger)
	cur := cursor.NewCursor()
	msg := contents.NewMessage("")
	scr := screen.NewScreen(mockBuilder, mockWriter, msg, cur, 24, 80)

	controller := NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, mockEventBus)

	// 操作によって呼び出される RefreshScreen の共通モックをセットアップする
	mockFileManager.EXPECT().GetFilename().Return("test.txt").AnyTimes()
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Clear().AnyTimes()
	mockBuilder.EXPECT().Write(gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Build().Return("").AnyTimes()
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()

	t.Run("InsertChar", func(t *testing.T) {
		// コンテンツをリセットする
		c = contents.NewContents(mockLogger)
		controller = NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, mockEventBus)
		// 必要に応じて依存関係を再注入するか、参照を保持している場合はコントローラーをそのまま使用する

		// プライベートメソッドにアクセスするか、Process 経由でトリガーする必要がある。
		// 特定の操作をテストしたいがそれらはプライベートであるため、
		// 特定のイベントを使用してパブリックな Process メソッド経由でテストできる。

		// 'a' キー押下をシミュレートする
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventChar, Rune: 'a'}, nil, nil)
		err := controller.Process()
		assert.NoError(t, err)
		assert.Equal(t, "a", c.GetAllLines()[0])
		assert.Equal(t, 1, cur.Col())
	})

	t.Run("InsertNewline", func(t *testing.T) {
		// Enter キーをシミュレートする
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyEnter}, nil, nil)
		err := controller.Process()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(c.GetAllLines()))
		assert.Equal(t, 0, cur.Col())
		assert.Equal(t, 1, cur.Row())
	})

	t.Run("DeleteChar", func(t *testing.T) {
		// Backspace をシミュレートする
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyBackspace}, nil, nil)
		err := controller.Process()
		assert.NoError(t, err)
		// 行が結合されるべき
		assert.Equal(t, 1, len(c.GetAllLines()))
		assert.Equal(t, "a", c.GetAllLines()[0])
	})

	t.Run("MoveCursor", func(t *testing.T) {
		// 左に移動
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyArrowLeft}, nil, nil)
		err := controller.Process()
		assert.NoError(t, err)
		assert.Equal(t, 0, cur.Col())

		// 右に移動
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyArrowRight}, nil, nil)
		err = controller.Process()
		assert.NoError(t, err)
		assert.Equal(t, 1, cur.Col())
	})
}

func TestController_Prompt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFileManager := mock_filemanager.NewMockFileManager(ctrl)
	mockInputProvider := mock_input.NewMockProvider(ctrl)
	mockLogger := mock_core.NewMockLogger(ctrl)
	mockWriter := mock_writer.NewMockScreenWriter(ctrl)
	mockBuilder := mock_contents.NewMockBuilder(ctrl)
	mockEventBus := event.NewBus()

	c := contents.NewContents(mockLogger)
	cur := cursor.NewCursor()
	msg := contents.NewMessage("")
	scr := screen.NewScreen(mockBuilder, mockWriter, msg, cur, 24, 80)

	controller := NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, mockEventBus)

	// プロンプト更新中に呼び出される RefreshScreen のモック
	mockFileManager.EXPECT().GetFilename().Return("").AnyTimes()
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Clear().AnyTimes()
	mockBuilder.EXPECT().Write(gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Build().Return("").AnyTimes()
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()

	// "test" + Enter の入力シーケンスをモックする
	// gomock.InOrder を使用するか、順次返す必要がある
	// ファイル名が空の場合、Ctrl+S（名前を付けて保存）経由でプロンプトをトリガーする
	// プライベートな prompt メソッドをトリガーする必要がある。
	// ファイル名が空のときに Ctrl+S で Process を呼び出すことでトリガーできる。

	// プロンプトのループは、上記で定義された inputEvents を消費する。
	// 注: Process -> createCommand -> createControlKeyCommand -> prompt と呼び出される
	// prompt は readEvent -> GetInputEvents を呼び出す

	// GetInputEvents 呼び出しの順序を保証する必要がある:
	// 1. Ctrl+S (Process 内)
	// 2. t, e, s, t, Enter (prompt 内)

	// 期待値を明確にするために再定義する
	gomock.InOrder(
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventControl, Key: key.KeyCtrlS}, nil, nil),
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventChar, Rune: 't'}, nil, nil),
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventChar, Rune: 'e'}, nil, nil),
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventChar, Rune: 's'}, nil, nil),
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventChar, Rune: 't'}, nil, nil),
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyEnter}, nil, nil),
	)

	// コントローラーのハンドラーによって SaveFile が呼び出されることを期待する
	mockFileManager.EXPECT().SaveFile("test", gomock.Any()).Return(nil).AnyTimes()

	// SaveEvent が発行されることを期待する
	// 検証するためにイベントバスを購読できる
	saveEventCh := make(chan struct{})
	mockEventBus.Subscribe(event.NewSingleTypeHandler(event.TypeSave, func(e event.Event) (bool, error) {
		if se, ok := e.Payload.(event.SaveEvent); ok {
			if se.Filename == "test" {
				close(saveEventCh)
			}
		}
		return true, nil
	}))

	err := controller.Process()
	assert.NoError(t, err)

	select {
	case <-saveEventCh:
		// 成功
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for save event")
	}
}

func TestController_MouseClick(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFileManager := mock_filemanager.NewMockFileManager(ctrl)
	mockInputProvider := mock_input.NewMockProvider(ctrl)
	mockLogger := mock_core.NewMockLogger(ctrl)
	mockWriter := mock_writer.NewMockScreenWriter(ctrl)
	mockBuilder := mock_contents.NewMockBuilder(ctrl)
	mockEventBus := event.NewBus()

	c := contents.NewContents(mockLogger)
	c.LoadContent([]string{"Hello", "World"})
	cur := cursor.NewCursor()
	msg := contents.NewMessage("")
	scr := screen.NewScreen(mockBuilder, mockWriter, msg, cur, 24, 80)

	controller := NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, mockEventBus)

	// RefreshScreen のモック
	mockFileManager.EXPECT().GetFilename().Return("test.txt").AnyTimes()
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Clear().AnyTimes()
	mockBuilder.EXPECT().Write(gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Build().Return("").AnyTimes()
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()

	// (1, 1) でのマウスクリックをシミュレート -> "World" の "o" (1行目、1列目)
	mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{
		Type:        key.KeyEventMouse,
		Key:         key.KeyMouseClick,
		MouseAction: key.MouseLeftClick,
		MouseRow:    1,
		MouseCol:    1,
	}, nil, nil)

	err := controller.Process()
	assert.NoError(t, err)

	// カーソル位置を検証する
	assert.Equal(t, 1, cur.Col())
	assert.Equal(t, 1, cur.Row())
}

func TestController_OpenFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFileManager := mock_filemanager.NewMockFileManager(ctrl)
	mockInputProvider := mock_input.NewMockProvider(ctrl)
	mockLogger := mock_core.NewMockLogger(ctrl)
	mockWriter := mock_writer.NewMockScreenWriter(ctrl)
	mockBuilder := mock_contents.NewMockBuilder(ctrl)
	mockEventBus := event.NewBus()

	c := contents.NewContents(mockLogger)
	cur := cursor.NewCursor()
	msg := contents.NewMessage("")
	scr := screen.NewScreen(mockBuilder, mockWriter, msg, cur, 24, 80)

	controller := NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, mockEventBus)

	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()
	mockFileManager.EXPECT().OpenFile("test.txt").Return(nil)
	mockFileManager.EXPECT().GetFilename().Return("test.txt").AnyTimes()

	err := controller.OpenFile("test.txt")
	assert.NoError(t, err)
}

func TestController_SpecialKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFileManager := mock_filemanager.NewMockFileManager(ctrl)
	mockInputProvider := mock_input.NewMockProvider(ctrl)
	mockLogger := mock_core.NewMockLogger(ctrl)
	mockWriter := mock_writer.NewMockScreenWriter(ctrl)
	mockBuilder := mock_contents.NewMockBuilder(ctrl)
	mockEventBus := event.NewBus()

	c := contents.NewContents(mockLogger)
	cur := cursor.NewCursor()
	msg := contents.NewMessage("")
	scr := screen.NewScreen(mockBuilder, mockWriter, msg, cur, 24, 80)

	controller := NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, mockEventBus)

	// RefreshScreen のモック
	mockFileManager.EXPECT().GetFilename().Return("test.txt").AnyTimes()
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Clear().AnyTimes()
	mockBuilder.EXPECT().Write(gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Build().Return("").AnyTimes()
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()

	t.Run("Tab", func(t *testing.T) {
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyTab}, nil, nil)
		err := controller.Process()
		assert.NoError(t, err)
		// Tab はデフォルトでスペース4つを挿入する
		assert.Equal(t, "    ", c.GetAllLines()[0])
		assert.Equal(t, 4, cur.Col())
	})

	t.Run("ShiftTab", func(t *testing.T) {
		// インデント付きのコンテンツをリセットしてセットアップする
		c = contents.NewContents(mockLogger)
		c.LoadContent([]string{"    indent"})
		cur = cursor.NewCursor()
		cur.SetCursor(4, 0) // インデント後のカーソル
		scr = screen.NewScreen(mockBuilder, mockWriter, msg, cur, 24, 80)
		controller = NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, mockEventBus)

		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyShiftTab}, nil, nil)
		err := controller.Process()
		assert.NoError(t, err)
		// ShiftTab はスペース4つを削除する
		assert.Equal(t, "indent", c.GetAllLines()[0])
		assert.Equal(t, 0, cur.Col())
	})
}

func TestController_MouseWheel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFileManager := mock_filemanager.NewMockFileManager(ctrl)
	mockInputProvider := mock_input.NewMockProvider(ctrl)
	mockLogger := mock_core.NewMockLogger(ctrl)
	mockWriter := mock_writer.NewMockScreenWriter(ctrl)
	mockBuilder := mock_contents.NewMockBuilder(ctrl)
	mockEventBus := event.NewBus()

	c := contents.NewContents(mockLogger)
	// スクロールするのに十分な行を追加する
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "line"
	}
	c.LoadContent(lines)

	cur := cursor.NewCursor()
	msg := contents.NewMessage("")
	scr := screen.NewScreen(mockBuilder, mockWriter, msg, cur, 24, 80)

	controller := NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, mockEventBus)

	// RefreshScreen のモック
	mockFileManager.EXPECT().GetFilename().Return("test.txt").AnyTimes()
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Clear().AnyTimes()
	mockBuilder.EXPECT().Write(gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Build().Return("").AnyTimes()
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()

	t.Run("ScrollDown", func(t *testing.T) {
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{
			Type:        key.KeyEventMouse,
			Key:         key.KeyMouseWheel,
			MouseAction: key.MouseScrollDown,
		}, nil, nil)

		err := controller.Process()
		assert.NoError(t, err)
		// カーソルは下に移動するべき（実装詳細: 3行）
		assert.Equal(t, 3, cur.Row())
	})

	t.Run("ScrollUp", func(t *testing.T) {
		mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{
			Type:        key.KeyEventMouse,
			Key:         key.KeyMouseWheel,
			MouseAction: key.MouseScrollUp,
		}, nil, nil)

		err := controller.Process()
		assert.NoError(t, err)
		// カーソルは上に移動するべき（0に戻る）
		assert.Equal(t, 0, cur.Row())
	})
}

func TestController_Quit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFileManager := mock_filemanager.NewMockFileManager(ctrl)
	mockInputProvider := mock_input.NewMockProvider(ctrl)
	mockLogger := mock_core.NewMockLogger(ctrl)
	mockWriter := mock_writer.NewMockScreenWriter(ctrl)
	mockBuilder := mock_contents.NewMockBuilder(ctrl)
	mockEventBus := event.NewBus()

	c := contents.NewContents(mockLogger)
	cur := cursor.NewCursor()
	msg := contents.NewMessage("")
	scr := screen.NewScreen(mockBuilder, mockWriter, msg, cur, 24, 80)

	controller := NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, mockEventBus)

	// RefreshScreen のモック
	mockFileManager.EXPECT().GetFilename().Return("test.txt").AnyTimes()
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Clear().AnyTimes()
	mockBuilder.EXPECT().Write(gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Build().Return("").AnyTimes()
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()

	// QuitEvent が発行されることを期待する
	quitEventCh := make(chan struct{})
	mockEventBus.Subscribe(event.NewSingleTypeHandler(event.TypeQuit, func(e event.Event) (bool, error) {
		close(quitEventCh)
		return true, nil
	}))

	mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventControl, Key: key.KeyCtrlX}, nil, nil)

	err := controller.Process()
	assert.NoError(t, err)

	select {
	case <-quitEventCh:
		// 成功
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for quit event")
	}
}

func TestController_InputBuffer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFileManager := mock_filemanager.NewMockFileManager(ctrl)
	mockInputProvider := mock_input.NewMockProvider(ctrl)
	mockLogger := mock_core.NewMockLogger(ctrl)
	mockWriter := mock_writer.NewMockScreenWriter(ctrl)
	mockBuilder := mock_contents.NewMockBuilder(ctrl)
	mockEventBus := event.NewBus()

	c := contents.NewContents(mockLogger)
	cur := cursor.NewCursor()
	msg := contents.NewMessage("")
	scr := screen.NewScreen(mockBuilder, mockWriter, msg, cur, 24, 80)

	controller := NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, mockEventBus)

	// RefreshScreen のモック
	mockFileManager.EXPECT().GetFilename().Return("test.txt").AnyTimes()
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Clear().AnyTimes()
	mockBuilder.EXPECT().Write(gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Build().Return("").AnyTimes()
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()

	// 'a' を返し、'b' をバッファリングする
	mockInputProvider.EXPECT().GetInputEvents().Return(
		key.KeyEvent{Type: key.KeyEventChar, Rune: 'a'},
		[]key.KeyEvent{{Type: key.KeyEventChar, Rune: 'b'}},
		nil,
	)

	// 最初の Process 呼び出しは 'a' を消費し、'b' をバッファリングする
	err := controller.Process()
	assert.NoError(t, err)
	assert.Equal(t, "a", c.GetAllLines()[0])

	// 2回目の Process 呼び出しはバッファから 'b' を消費する（GetInputEvents は呼び出されない）
	err = controller.Process()
	assert.NoError(t, err)
	assert.Equal(t, "ab", c.GetAllLines()[0])
}
