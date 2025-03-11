package editor

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"

	"github.com/wasya-io/go-kilo/app/boundary/provider/input"
	"github.com/wasya-io/go-kilo/app/config"
	"github.com/wasya-io/go-kilo/app/entity/contents"
	"github.com/wasya-io/go-kilo/app/entity/core"
	"github.com/wasya-io/go-kilo/app/entity/core/term"
	"github.com/wasya-io/go-kilo/app/entity/cursor"
	"github.com/wasya-io/go-kilo/app/entity/key"
	"github.com/wasya-io/go-kilo/app/entity/screen"
	"github.com/wasya-io/go-kilo/app/usecase/controller"
	"github.com/wasya-io/go-kilo/editor/events"
)

// Editor はエディタの状態を管理する構造体
type Editor struct {
	term              *term.TerminalState
	screen            screen.Screen
	controller        *controller.Controller
	isQuitting        bool
	quitWarningShown  bool
	buffer            *contents.Contents
	eventBuffer       []key.KeyEvent
	config            *config.Config
	eventManager      *events.EventManager // 追加: イベントマネージャー
	termState         *term.TerminalState
	cleanupOnce       sync.Once
	cleanupChan       chan struct{}
	logger            core.Logger
	statusMessage     string
	statusMessageTime int
	stateManager      *EditorStateManager // 追加
	inputProvider     input.Provider      // 追加
}

type WinSize struct {
	Rows int
	Cols int
}

// New は新しいEditorインスタンスを作成する
func New(
	testMode bool,
	conf *config.Config,
	logger core.Logger,
	eventManager *events.EventManager,
	buffer *contents.Contents,
	inputProvider input.Provider,
	screen screen.Screen,
	controller *controller.Controller,
) (*Editor, error) {
	// 1. 必須コンポーネントのチェック
	if eventManager == nil || buffer == nil {
		return nil, fmt.Errorf("required components are not initialized")
	}

	// 6. Editorインスタンスの作成
	e := &Editor{
		screen:           screen,
		controller:       controller,
		buffer:           buffer,
		config:           conf,
		eventManager:     eventManager,
		isQuitting:       false,
		quitWarningShown: false,
		cleanupChan:      make(chan struct{}),
		logger:           logger,
		inputProvider:    inputProvider,
	}

	if !testMode {
		// 8. 初期コンテンツの設定
		defaultContent := []string{
			"Hello, Go-Kilo editor!",
			"Use arrow keys to move cursor.",
			"Press Ctrl-Q or Ctrl-C to quit.",
		}
		e.buffer.LoadContent(defaultContent)

		// 9. ターミナルの設定
		term, err := term.EnableRawMode()
		if err != nil {
			return nil, err
		}
		e.term = term
		e.termState = term

		// 10. クリーンアップハンドラの設定
		go e.setupCleanupHandler()
	}

	return e, nil
}

// setupCleanupHandler はクリーンアップハンドラをセットアップする
func (e *Editor) setupCleanupHandler() {
	defer func() {
		if r := recover(); r != nil {
			// パニック時の端末状態復元を保証
			e.Cleanup()
			// スタックトレースとエラー情報を出力
			fmt.Fprintf(os.Stderr, "Editor panic: %v\n", r)
			debug.PrintStack()
			os.Exit(1)
		}
	}()

	// シグナル処理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigChan:
		e.Cleanup()
		os.Exit(0)
	case <-e.cleanupChan:
		return
	}
}

// Cleanup は終了時の後処理を行う
func (e *Editor) Cleanup() {
	e.cleanupOnce.Do(func() {
		// 最後にログをフラッシュする
		e.logger.Flush()

		// 端末の状態を復元
		if e.termState != nil {
			e.termState.DisableRawMode()
			e.termState = nil
		}

		// クリーンアップ処理の完了を通知
		close(e.cleanupChan)

		// その他のクリーンアップ処理
		os.Stdout.WriteString(e.screen.ClearScreen())
		os.Stdout.WriteString(e.screen.MoveCursorToHome())
	})
}

// Quit はエディタを終了する
func (e *Editor) Quit() {
	e.Cleanup()
	os.Exit(0)
}

// IsQuitWarningShown は終了警告が表示されているかを返す
func (e *Editor) IsQuitWarningShown() bool {
	return e.quitWarningShown
}

// SetQuitWarningShown は終了警告の表示状態を設定する
func (e *Editor) SetQuitWarningShown(shown bool) {
	e.quitWarningShown = shown
}

// EditorOperationsインターフェースの実装
func (e *Editor) GetConfig() *config.Config {
	return e.config
}

func (e *Editor) IsDirty() bool {
	return e.buffer.IsDirty()
}

func (e *Editor) SetDirty(dirty bool) {
	e.buffer.SetDirty(dirty)
}

// GetCursor はUI経由でカーソル位置を返す
func (e *Editor) GetCursor() *cursor.Cursor {
	return e.screen.GetCursor()
}

func (e *Editor) GetContent(lineNum int) string {
	return e.buffer.GetContentLine(lineNum)
}

// Run はエディタのメインループを実行する
func (e *Editor) Run() error {
	defer e.Cleanup()

	e.logger.Log("system", "Editor starting")
	defer e.logger.Log("system", "Editor shutting down")

	// 初期表示
	if err := e.controller.RefreshScreen(); err != nil {
		return err
	}

	for {
		select {
		case <-e.controller.Quit:
			return nil
		default:
			if err := e.controller.Process(); err != nil {
				e.logger.Log("error", fmt.Sprintf("Main loop error: %v", err))
				return err
			}
		}
	}
}

// GetEventManager はEventManagerを返す
func (e *Editor) GetEventManager() *events.EventManager {
	return e.eventManager
}
