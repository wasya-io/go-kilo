package editor

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/wasya-io/go-kilo/editor/events"
	"github.com/wasya-io/go-kilo/editor/logger"
	"golang.org/x/sys/unix"
)

// Editor はエディタの状態を管理する構造体
type Editor struct {
	term         *terminalState
	ui           *UI
	quit         chan struct{}
	isQuitting   bool
	buffer       *Buffer
	rowOffset    int
	colOffset    int
	fileManager  *FileManager
	input        *InputHandler
	config       *Config
	eventManager *events.EventManager // 追加: イベントマネージャー
	termState    *terminalState
	cleanupOnce  sync.Once
	cleanupChan  chan struct{}
	logger       *logger.Logger
}

// New は新しいEditorインスタンスを作成する
func New(testMode bool) (*Editor, error) {
	var ws *unix.Winsize
	var err error
	if !testMode {
		ws, err = unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
		if err != nil {
			return nil, err
		}
	} else {
		ws = &unix.Winsize{Row: 24, Col: 80}
	}

	screenRows := int(ws.Row)
	screenCols := int(ws.Col)

	eventManager := events.NewEventManager()
	config := LoadConfig()

	e := &Editor{
		ui:           NewUI(screenRows, screenCols, eventManager), // eventManagerを追加
		quit:         make(chan struct{}),
		buffer:       NewBuffer(eventManager), // eventManagerを引数として渡す
		rowOffset:    0,
		colOffset:    0,
		config:       config,
		eventManager: eventManager,
		isQuitting:   false,
		cleanupChan:  make(chan struct{}),
		logger:       logger.New(config.DebugMode),
	}

	e.fileManager = NewFileManager(e.buffer, eventManager)
	e.input = NewInputHandler(e, eventManager)

	// イベントハンドラの登録
	e.setupEventHandlers()

	if !testMode {
		defaultContent := []string{
			"Hello, Go-Kilo editor!",
			"Use arrow keys to move cursor.",
			"Press Ctrl-Q or Ctrl-C to quit.",
		}
		e.buffer.LoadContent(defaultContent)

		term, err := enableRawMode()
		if err != nil {
			return nil, err
		}
		e.term = term
		e.termState = term

		// パニックリカバリーとクリーンアップの設定
		// TODO: main.goと似たようなことを行なっている->ここでもリカバリや終了シグナル待ちを行なっているのでgoroutineリークしているのでは？
		go func() {
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
		}()
	}

	return e, nil
}

// setupRecoveryHandlers はリカバリーハンドラを設定する
func (e *Editor) setupRecoveryHandlers() {
	// パニックハンドラを設定
	defer func() {
		if r := recover(); r != nil {
			e.handlePanic(r)
		}
	}()

	// バッファイベントのリカバリー戦略を設定
	e.eventManager.SetRecoveryStrategy(events.RollbackToStable)

	// クリティカルエラー用のグローバルハンドラを設定
	e.eventManager.SetGlobalErrorHandler(func(err error) {
		stack := debug.Stack()
		e.setStatusMessage("Critical error: %v", err)

		// スタックトレースとエラー情報を一時ファイルに保存
		timestamp := time.Now().Format("20060102-150405")
		errorFile := fmt.Sprintf("error-%s.log", timestamp)
		content := []string{
			fmt.Sprintf("Error: %v", err),
			"Stack trace:",
			string(stack),
		}
		e.fileManager.SaveFile(errorFile, content)

		// 自動保存を試行
		if e.buffer != nil && e.buffer.IsDirty() {
			e.saveBufferToTempFile()
		}
	})
}

// handlePanic はパニックから復帰を試みる
func (e *Editor) handlePanic(r interface{}) {
	// パニック情報を記録
	stack := debug.Stack()
	timestamp := time.Now().Format("20060102-150405")
	crashFile := fmt.Sprintf("crash-%s.log", timestamp)
	content := []string{
		fmt.Sprintf("Panic: %v", r),
		"Stack trace:",
		string(stack),
	}
	e.fileManager.SaveFile(crashFile, content)

	// バッファの保存を試みる
	if e.buffer != nil && e.buffer.IsDirty() {
		e.saveBufferToTempFile()
	}

	// 最後の安定状態への復帰を試みる
	if err := e.RecoverFromLatestSnapshot(); err != nil {
		e.setStatusMessage("Failed to recover: %v", err)
		return
	}

	e.setStatusMessage("Recovered from panic. Crash log saved to %s", crashFile)
}

// setupEventHandlers はイベントハンドラを設定する
func (e *Editor) setupEventHandlers() {
	// リカバリーハンドラを設定
	e.setupRecoveryHandlers()

	// バッファイベントのハンドラを登録
	e.eventManager.Subscribe(events.BufferEventType, func(event events.Event) {
		if bufferEvent, ok := event.(*events.BufferEvent); ok {
			e.handleBufferEvent(bufferEvent)
		}
	})

	// UIイベントのハンドラを登録
	e.eventManager.Subscribe(events.UIEventType, func(event events.Event) {
		if uiEvent, ok := event.(*events.UIEvent); ok {
			e.handleUIEvent(uiEvent)
		}
	})

	// ファイルイベントのハンドラを登録
	e.eventManager.Subscribe(events.FileEventType, func(event events.Event) {
		if fileEvent, ok := event.(*events.FileEvent); ok {
			e.handleFileEvent(fileEvent)
		}
	})

	// エラーハンドラの設定
	e.eventManager.SetGlobalErrorHandler(func(err error) {
		e.setStatusMessage("Error: %v", err)
		// 自動保存を試行
		if e.buffer != nil && e.buffer.IsDirty() {
			e.saveBufferToTempFile()
		}
	})

	// バッファイベントのエラーハンドラ
	e.eventManager.SetErrorHandler(events.BufferEventType, func(event events.Event, err error) {
		if bufferEvent, ok := event.(*events.BufferEvent); ok {
			// 前回の状態に復帰を試みる
			prevState, _ := bufferEvent.GetStates()
			e.buffer.RestoreState(prevState)
			e.setStatusMessage("Recovered from error: %v", err)
		}
	})

	// 定期的なスナップショット作成
	go e.periodicSnapshot()

	// エラー統計の定期チェック
	go e.monitorErrors()
}

// handleBufferEvent はバッファイベントを処理する
func (e *Editor) handleBufferEvent(event *events.BufferEvent) {
	e.ui.BeginBatchUpdate()
	defer e.ui.EndBatchUpdate()

	switch event.SubType {
	case events.BufferContentChanged:
		if data, ok := event.Data.(string); ok {
			e.UpdateScroll()
			// テキスト変更はすべて再描画が必要
			e.ui.QueueUpdate(AreaFull, MediumPriority, data)
		}
	case events.BufferCursorMoved:
		if data, ok := event.Data.(events.Position); ok {
			e.UpdateScroll()
			// カーソル移動時は部分更新のみ
			e.ui.QueueUpdate(AreaCursor, HighPriority, data)
		}
	case events.BufferStructuralChange:
		e.UpdateScroll()
		e.ui.QueueUpdate(AreaFull, MediumPriority, nil)
	}

	// ステータスの更新も必要な場合
	if event.HasChanges() {
		e.ui.QueueUpdate(AreaStatus, LowPriority, nil)
	}
}

// handleUIEvent はUIイベントを処理する
func (e *Editor) handleUIEvent(event *events.UIEvent) {
	switch event.SubType {
	case events.UIRefresh:
		e.RefreshScreen()
	case events.UIScroll:
		if data, ok := event.Data.(events.ScrollData); ok {
			e.ui.handleScrollEvent(data)
			e.RefreshScreen()
		}
	case events.UIStatusMessage:
		if data, ok := event.Data.(events.StatusMessageData); ok {
			e.ui.SetMessage(data.Message, data.Args...)
			e.RefreshScreen()
		}
	case events.UICursorUpdate:
		if data, ok := event.Data.(events.Position); ok {
			e.buffer.SetCursor(data.X, data.Y)
			e.RefreshScreen()
		}
	}
}

// handleFileEvent はファイルイベントを処理する
func (e *Editor) handleFileEvent(event *events.FileEvent) {
	switch event.SubType {
	case events.FileOpen:
		if event.HasError() {
			e.setStatusMessage("Error opening file: %v", event.GetError())
		} else {
			e.buffer.LoadContent(event.GetContent())
			e.setStatusMessage("File loaded: %s", event.GetFilename())
		}
	case events.FileSave:
		if event.HasError() {
			e.setStatusMessage("Error saving file: %v", event.GetError())
		} else {
			e.buffer.SetDirty(false)
			e.setStatusMessage("File saved: %s", event.GetFilename())
		}
	}
}

// publishUIEvent はUIイベントを発行する
func (e *Editor) publishUIEvent(subType events.UIEventSubType, data interface{}) {
	if e.eventManager != nil {
		event := events.NewUIEvent(subType, data)
		e.eventManager.Publish(event)
	}
}

// Cleanup は終了時の後処理を行う
func (e *Editor) Cleanup() {
	e.cleanupOnce.Do(func() {
		// 最後にログをフラッシュする
		e.logger.Flush()

		// 端末の状態を復元
		if e.termState != nil {
			e.termState.disableRawMode()
			e.termState = nil
		}

		// クリーンアップ処理の完了を通知
		close(e.cleanupChan)

		// その他のクリーンアップ処理
		os.Stdout.WriteString(e.ui.clearScreen())
		os.Stdout.WriteString(e.ui.moveCursorToHome())
	})
}

// RefreshScreen は画面を更新する
func (e *Editor) RefreshScreen() error {
	// UI更新の前にスクロール位置を更新
	e.UpdateScroll()

	// UIの更新処理を実行
	err := e.ui.RefreshScreen(e.buffer, e.fileManager.GetFilename(), e.rowOffset, e.colOffset)
	if err != nil {
		return err
	}

	// メッセージ表示後は即座にフラッシュする
	return e.ui.Flush()
}

// ProcessKeypress はキー入力を処理する
func (e *Editor) ProcessKeypress() error {
	command, err := e.input.HandleKeypress()
	if err != nil {
		e.logger.Log("error", fmt.Sprintf("Keypress error: %v", err))
		return err
	}

	// 画面更新を必ず行う（コマンドの有無に関わらず）
	defer e.RefreshScreen()

	if command != nil {
		// コマンドを実行
		e.logger.Log("command", fmt.Sprintf("Executed command: %T", command))
		if err := command.Execute(); err != nil {
			return err
		}
	}

	return nil
}

// UpdateScroll はカーソル位置に基づいてスクロール位置を更新する
func (e *Editor) UpdateScroll() {
	// スクロール位置の更新処理
	if e.buffer.cursor.Y < e.rowOffset {
		e.rowOffset = e.buffer.cursor.Y
	}

	screenBottom := e.ui.screenRows - 2
	visibleLines := screenBottom - 1

	if e.buffer.cursor.Y >= e.rowOffset+visibleLines {
		e.rowOffset = e.buffer.cursor.Y - visibleLines + 1
	}

	row := e.buffer.getRow(e.buffer.cursor.Y)
	if row == nil {
		return
	}

	cursorScreenPos := row.OffsetToScreenPosition(e.buffer.cursor.X)

	if cursorScreenPos < e.colOffset {
		e.colOffset = cursorScreenPos
	}

	rightMargin := (e.ui.screenCols * 4) / 5
	if cursorScreenPos >= e.colOffset+rightMargin {
		e.colOffset = cursorScreenPos - rightMargin + 1
	}

	if e.rowOffset < 0 {
		e.rowOffset = 0
	}
	if e.colOffset < 0 {
		e.colOffset = 0
	}

	maxRow := max(0, e.buffer.GetLineCount()-1)
	if e.rowOffset > maxRow {
		e.rowOffset = maxRow
	}
}

// QuitChan は終了シグナルを監視するためのチャネルを返す
func (e *Editor) QuitChan() <-chan struct{} {
	return e.quit
}

// Quit はエディタを終了する
func (e *Editor) Quit() {
	e.Cleanup()
	os.Exit(0)
}

// OpenFile は指定されたファイルを読み込む
func (e *Editor) OpenFile(filename string) error {
	e.logger.Log("file", fmt.Sprintf("Opening file: %s", filename))

	if err := e.fileManager.OpenFile(filename); err != nil {
		e.logger.Log("error", fmt.Sprintf("Failed to open file: %s, error: %v", filename, err))
		return err
	}
	e.setStatusMessage("File loaded")
	return nil
}

// SaveFile は現在の内容をファイルに保存する
func (e *Editor) SaveFile() error {
	e.logger.Log("file", "Saving current file")

	if err := e.fileManager.SaveCurrentFile(); err != nil {
		if err == ErrNoFilename {
			e.logger.Log("error", "No filename specified for save")
			e.setStatusMessage("No filename")
			return nil
		}
		e.logger.Log("error", fmt.Sprintf("Failed to save file: %v", err))
		return err
	}
	e.setStatusMessage("File saved")
	return nil
}

// setStatusMessage はステータスメッセージを設定する（非公開メソッド）
func (e *Editor) setStatusMessage(format string, args ...interface{}) {
	if e.config.DebugMode {
		format = "[in Debug] " + format
	}
	// UIコンポーネントのSetMessageメソッドを呼び出す
	e.ui.SetMessage(format, args...)

	// 即座に画面を更新して変更を反映
	e.RefreshScreen()
}

// SetStatusMessage はステータスメッセージを設定する（EditorOperations用の公開メソッド）
func (e *Editor) SetStatusMessage(format string, args ...interface{}) {
	e.setStatusMessage(format, args...)
}

// EditorOperationsインターフェースの実装
func (e *Editor) GetConfig() *Config {
	return e.config
}

func (e *Editor) InsertChar(ch rune) {
	e.logger.Log("edit", fmt.Sprintf("Inserting character: %c", ch))
	e.buffer.InsertChar(ch)
	e.RefreshScreen()
}

func (e *Editor) DeleteChar() {
	e.logger.Log("edit", "Deleting character")
	e.buffer.DeleteChar()
	e.RefreshScreen()
}

func (e *Editor) MoveCursor(movement CursorMovement) {
	e.logger.Log("cursor", fmt.Sprintf("Moving cursor: %v", movement))
	e.buffer.MoveCursor(movement)
	e.UpdateScroll()
}

func (e *Editor) InsertNewline() {
	e.logger.Log("edit", "Inserting newline")
	e.buffer.InsertNewline()
	e.RefreshScreen()
}

func (e *Editor) IsDirty() bool {
	return e.buffer.IsDirty()
}

func (e *Editor) SetDirty(dirty bool) {
	e.buffer.SetDirty(dirty)
}

func (e *Editor) GetCursor() Cursor {
	return e.buffer.GetCursor()
}

func (e *Editor) GetContent(lineNum int) string {
	return e.buffer.GetContentLine(lineNum)
}

func (e *Editor) InsertChars(chars []rune) {
	e.buffer.InsertChars(chars)
}

// periodicSnapshot は定期的にスナップショットを作成する
func (e *Editor) periodicSnapshot() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if e.buffer != nil && e.buffer.IsDirty() {
				e.eventManager.CreateSnapshot()
			}
		case <-e.quit:
			return
		}
	}
}

// saveBufferToTempFile はバッファの内容を一時ファイルに保存する
func (e *Editor) saveBufferToTempFile() {
	baseName := "untitled"
	if e.buffer.Filename != "" {
		baseName = e.buffer.Filename
	}
	tempFile := fmt.Sprintf("%s.recovery", baseName)
	if err := e.fileManager.SaveFile(tempFile, e.buffer.GetAllLines()); err != nil {
		e.setStatusMessage("Failed to create recovery file: %v", err)
		return
	}
	e.setStatusMessage("Recovery file created: %s", tempFile)
}

// RecoverFromLatestSnapshot は最新のスナップショットから復元を試みる
func (e *Editor) RecoverFromLatestSnapshot() error {
	return e.eventManager.RecoverFromSnapshot(time.Now())
}

// monitorErrors はエラー統計を定期的にチェックする
func (e *Editor) monitorErrors() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := e.eventManager.GetErrorStats()
			if criticalErrors := stats[events.BufferEventType] + stats[events.FileEventType]; criticalErrors > 0 {
				e.setStatusMessage("Warning: %d critical errors in last 5 minutes", criticalErrors)
			}
		case <-e.quit:
			return
		}
	}
}

// Run はエディタのメインループを実行する
func (e *Editor) Run() error {
	defer e.Cleanup()

	e.logger.Log("system", "Editor starting")
	defer e.logger.Log("system", "Editor shutting down")

	// 初期表示
	if err := e.RefreshScreen(); err != nil {
		return err
	}

	for {
		select {
		case <-e.quit:
			return nil
		default:
			if err := e.ProcessKeypress(); err != nil {
				e.logger.Log("error", fmt.Sprintf("Main loop error: %v", err))
				return err
			}
		}
	}
}
