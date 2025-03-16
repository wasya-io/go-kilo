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
	"github.com/wasya-io/go-kilo/app/entity/event"
	"github.com/wasya-io/go-kilo/app/entity/screen"
	"github.com/wasya-io/go-kilo/app/usecase/controller"
)

// Editor はエディタの状態を管理する構造体
type Editor struct {
	term             *term.TerminalState
	screen           *screen.Screen
	controller       *controller.Controller
	isQuitting       bool
	quitWarningShown bool
	buffer           *contents.Contents
	config           *config.Config
	termState        *term.TerminalState
	cleanupOnce      sync.Once
	cleanupChan      chan struct{}
	logger           core.Logger
	inputProvider    input.Provider
	eventBus         *event.Bus // イベントバスを追加
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
	buffer *contents.Contents,
	inputProvider input.Provider,
	screen *screen.Screen,
	controller *controller.Controller,
	eventBus *event.Bus, // イベントバスを追加
) (*Editor, error) {
	// 6. Editorインスタンスの作成
	e := &Editor{
		screen:           screen,
		controller:       controller,
		buffer:           buffer,
		config:           conf,
		isQuitting:       false,
		quitWarningShown: false,
		cleanupChan:      make(chan struct{}),
		logger:           logger,
		inputProvider:    inputProvider,
		eventBus:         eventBus, // イベントバスを設定
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
		// イベントバスのシャットダウン
		if e.eventBus != nil {
			e.eventBus.Shutdown()
		}

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

func (e *Editor) OpenFile(filename string) error {
	return e.controller.OpenFile(filename)
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
