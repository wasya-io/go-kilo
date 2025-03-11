package editor

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/wasya-io/go-kilo/app/boundary/provider/input"
	"github.com/wasya-io/go-kilo/app/config"
	"github.com/wasya-io/go-kilo/app/entity/contents"
	"github.com/wasya-io/go-kilo/app/entity/core"
	"github.com/wasya-io/go-kilo/app/entity/core/term"
	"github.com/wasya-io/go-kilo/app/entity/cursor"
	"github.com/wasya-io/go-kilo/app/entity/key"
	"github.com/wasya-io/go-kilo/app/entity/screen"
	"github.com/wasya-io/go-kilo/editor/events"
)

// Editor はエディタの状態を管理する構造体
type Editor struct {
	term *term.TerminalState
	// ui               *UI
	screen           screen.Screen
	quit             chan struct{}
	isQuitting       bool
	quitWarningShown bool
	buffer           *contents.Contents
	eventBuffer      []key.KeyEvent
	fileManager      *FileManager
	// input             *InputHandler
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
	fileManager *FileManager,
	inputProvider input.Provider,
	screen screen.Screen,
) (*Editor, error) {
	// 1. 必須コンポーネントのチェック
	if eventManager == nil || buffer == nil || fileManager == nil {
		return nil, fmt.Errorf("required components are not initialized")
	}

	// 6. Editorインスタンスの作成
	e := &Editor{
		screen:           screen,
		quit:             make(chan struct{}),
		buffer:           buffer,
		config:           conf,
		eventManager:     eventManager,
		isQuitting:       false,
		quitWarningShown: false,
		cleanupChan:      make(chan struct{}),
		logger:           logger,
		fileManager:      fileManager,
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

	// 11. システムイベントハンドラの登録
	eventManager.RegisterSystemEventHandler(e)

	return e, nil
}

// SetStateManager はStateManagerを設定する
func (e *Editor) SetStateManager(manager *EditorStateManager) {
	if manager == nil {
		panic("state manager cannot be nil")
	}
	e.stateManager = manager
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

		// スタックトレースとエラー情報をログとして保存
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
	// 1. パニック情報のログ記録
	stack := debug.Stack()
	timestamp := time.Now().Format("20060102-150405")
	crashFile := fmt.Sprintf("crash-%s.log", timestamp)
	content := []string{
		fmt.Sprintf("Panic: %v", r),
		"Stack trace:",
		string(stack),
	}
	e.fileManager.SaveFile(crashFile, content)

	// 2. バッファの保存を試みる
	if e.buffer != nil && e.buffer.IsDirty() {
		e.saveBufferToTempFile()
	}

	// 3. エラー状態の記録
	e.setStatusMessage("Recovered from panic. Crash log saved to %s", crashFile)

	// 4. 状態の復元を試みる
	if e.stateManager != nil {
		if err := e.stateManager.RecoverFromSnapshot(time.Now()); err != nil {
			// 復元に失敗した場合はエラーを記録するのみ
			e.logger.Log("error", fmt.Sprintf("Failed to recover state: %v", err))
		}
	}
}

// setupEventHandlers はイベントハンドラを設定する
func (e *Editor) setupEventHandlers() {
	// 1. コンポーネントの存在チェック
	if e.eventManager == nil {
		panic("event manager is not initialized")
	}
	if e.buffer == nil {
		panic("buffer is not initialized")
	}
	if e.stateManager == nil {
		panic("state manager is not initialized")
	}

	// 2. リカバリーハンドラを設定
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("failed to setup event handlers: %v", r))
		}
	}()

	e.setupRecoveryHandlers()

	// 3. バッファイベントのハンドラを登録
	e.eventManager.Subscribe(events.BufferEventType, func(event events.Event) {
		if bufferEvent, ok := event.(*events.BufferEvent); ok {
			e.handleBufferEvent(bufferEvent)
		}
	})

	// 4. UIイベントのハンドラを登録
	e.eventManager.Subscribe(events.UIEventType, func(event events.Event) {
		if uiEvent, ok := event.(*events.UIEvent); ok {
			e.handleUIEvent(uiEvent)
		}
	})

	// 5. ファイルイベントのハンドラを登録
	e.eventManager.Subscribe(events.FileEventType, func(event events.Event) {
		if fileEvent, ok := event.(*events.FileEvent); ok {
			e.handleFileEvent(fileEvent)
		}
	})

	// 6. エラーハンドラの設定
	e.eventManager.SetGlobalErrorHandler(func(err error) {
		e.setStatusMessage("Error: %v", err)
		if e.buffer != nil && e.buffer.IsDirty() {
			e.saveBufferToTempFile()
		}
	})

	// 7. バッファイベントのエラーハンドラ
	e.eventManager.SetErrorHandler(events.BufferEventType, func(event events.Event, err error) {
		if bufferEvent, ok := event.(*events.BufferEvent); ok {
			// 前回の状態に復帰を試みる
			prevState, _ := bufferEvent.GetStates()
			e.buffer.RestoreState(prevState)
			e.setStatusMessage("Recovered from error: %v", err)
		}
	})

	// 8. 状態管理の定期タスク設定
	if !e.isQuitting {
		go e.periodicSnapshot()
		go e.monitorErrors()
	}
}

// handleBufferEvent はバッファイベントを処理する
func (e *Editor) handleBufferEvent(event *events.BufferEvent) {
	// e.ui.BeginBatchUpdate()
	// defer e.ui.EndBatchUpdate()

	switch event.SubType {
	case events.BufferContentChanged:
		// 内容変更時の処理
		if event.HasChanges() {
		}
	case events.BufferStructuralChange:
		// 構造的な変更時の処理（改行など）
	}

}

// handleUIEvent はUIイベントを処理する
func (e *Editor) handleUIEvent(event *events.UIEvent) {
	switch event.SubType {
	case events.UIRefresh:
		e.RefreshScreen()
	case events.UIStatusMessage:
		if data, ok := event.Data.(events.StatusMessageData); ok {
			e.screen.SetMessage(data.Message, data.Args...)
			// e.ui.SetMessage(data.Message, data.Args...)
			e.RefreshScreen()
		}
	case events.UICursorUpdate:
		if data, ok := event.Data.(events.CursorPosition); ok {
			// カーソル位置の更新はUI構造体で管理
			e.screen.SetCursorPosition(data.X, data.Y)
			e.UpdateScroll()
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
// 現在未使用だが、将来的なUI更新の統合のために保持
/*
func (e *Editor) publishUIEvent(subType events.UIEventSubType, data interface{}) {
	if e.eventManager != nil {
		event := events.NewUIEvent(subType, data)
		e.eventManager.Publish(event)
	}
}
*/

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

// RefreshScreen は画面を更新する
func (e *Editor) RefreshScreen() error {
	// UI更新の前にスクロール位置を更新
	e.UpdateScroll()

	// UIの更新処理を実行
	colOffset, rowOffset := e.screen.GetOffset()
	err := e.screen.Redraw(e.buffer, e.fileManager.GetFilename(), rowOffset, colOffset)
	if err != nil {
		return err
	}

	// メッセージ表示後は即座にフラッシュする
	return e.screen.Flush()
}

// ProcessKeypress はキー入力を処理する
func (e *Editor) ProcessKeypress() error {
	event, err := e.readEvent()
	if err != nil {
		e.logger.Log("error", fmt.Sprintf("readEvent error: %v", err))
		return err
	}

	// コマンドを作成
	command, err := e.createCommand(event)
	if err != nil {
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

// readEvent はイベントを読み取る
func (e *Editor) readEvent() (key.KeyEvent, error) {

	// バッファにイベントがある場合はそれを返す
	if len(e.eventBuffer) > 0 {
		event := e.eventBuffer[0]
		e.eventBuffer = e.eventBuffer[1:]
		return event, nil
	}
	event, remainingEvents, err := e.inputProvider.GetInputEvents()
	if err != nil {
		e.logger.Log("error", fmt.Sprintf("Keypress error: %v", err))
		return key.KeyEvent{}, err
	}

	// 残りのイベントがある場合はバッファに追加
	if len(remainingEvents) > 0 {
		e.eventBuffer = append(e.eventBuffer, remainingEvents...)
	}
	return event, nil
}

// UpdateScroll はカーソル位置に基づいてスクロール位置を更新する
func (e *Editor) UpdateScroll() {
	// スクロール位置の更新処理
	offsetCol, offsetRow := e.screen.GetOffset()

	// UI経由でカーソル位置を取得
	pos := e.screen.GetCursor().ToPosition()
	row := e.buffer.GetRow(pos.Y)
	if row == nil {
		return
	}

	screenRowLines := e.screen.GetRowLines()
	// ステータスバーとメッセージバー用に2行確保
	visibleLines := screenRowLines - 2

	// スクロール条件の計算
	// カーソルが表示領域の上端より上にある場合
	if pos.Y < offsetRow {
		offsetRow = pos.Y
	}
	// カーソルが表示領域の下端に近づいた場合
	if pos.Y >= offsetRow+visibleLines {
		offsetRow = pos.Y - visibleLines + 1
	}

	// 水平方向のスクロール
	cursorScreenPos := row.OffsetToScreenPosition(pos.X)
	if cursorScreenPos < offsetCol {
		offsetCol = cursorScreenPos
	}

	screenColLines := e.screen.GetColLines()
	rightMargin := (screenColLines * 4) / 5
	if cursorScreenPos >= (offsetCol + rightMargin) {
		offsetCol = cursorScreenPos - rightMargin + 1
	}

	// スクロール位置の制限
	if offsetRow < 0 {
		offsetRow = 0
	}
	if offsetCol < 0 {
		offsetCol = 0
	}

	maxRow := e.buffer.GetLineCount() - 1
	if offsetRow > maxRow {
		offsetRow = maxRow
	}

	// スクロール位置の更新
	e.screen.SetRowOffset(offsetRow)
	e.screen.SetColOffset(offsetCol)
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
	return e.fileManager.OpenFile(filename)
}

// SaveFile は現在の内容をファイルに保存する
func (e *Editor) SaveFile() error {
	return e.fileManager.SaveCurrentFile()
}

// createCommand はキーイベントからコマンドを作成する
func (e *Editor) createCommand(event key.KeyEvent) (Command, error) {
	switch event.Type {
	case key.KeyEventChar, key.KeyEventSpecial:
		// 警告状態をクリア
		if e.quitWarningShown {
			e.quitWarningShown = false
			e.SetStatusMessage("")
		}
		if event.Type == key.KeyEventChar {
			return NewInsertCharCommand(e, event.Rune), nil
		}
		return e.createSpecialKeyCommand(event.Key), nil
	case key.KeyEventControl:
		return e.createControlKeyCommand(event.Key), nil
	case key.KeyEventMouse:
		if event.Key == key.KeyMouseWheel {
			// マウスホイールイベントは専用のカーソル移動コマンドを使用
			switch event.MouseAction {
			case key.MouseScrollUp:
				return NewMoveCursorCommand(e, cursor.MouseWheelUp), nil
			case key.MouseScrollDown:
				return NewMoveCursorCommand(e, cursor.MouseWheelDown), nil
			}
		} else if event.Key == key.KeyMouseClick {
			// マウスクリックイベントは現時点では無視
			// 必要に応じて適切なコマンドを実装できます
			return nil, nil
		}
	}
	return nil, nil
}

// createSpecialKeyCommand は特殊キーに対応するコマンドを作成する
func (e *Editor) createSpecialKeyCommand(k key.Key) Command {
	switch k {
	case key.KeyArrowLeft:
		return NewMoveCursorCommand(e, cursor.CursorLeft)
	case key.KeyArrowRight:
		return NewMoveCursorCommand(e, cursor.CursorRight)
	case key.KeyArrowUp:
		return NewMoveCursorCommand(e, cursor.CursorUp)
	case key.KeyArrowDown:
		return NewMoveCursorCommand(e, cursor.CursorDown)
	case key.KeyBackspace:
		return NewDeleteCharCommand(e)
	case key.KeyEnter:
		return NewInsertNewlineCommand(e)
	case key.KeyTab:
		return NewInsertTabCommand(e)
	case key.KeyShiftTab:
		return NewUndentCommand(e)
	default:
		return nil
	}
}

// createControlKeyCommand はコントロールキーに対応するコマンドを作成する
func (e *Editor) createControlKeyCommand(k key.Key) Command {
	switch k {
	case key.KeyCtrlS:
		return NewSaveCommand(e)
	case key.KeyCtrlQ, key.KeyCtrlC:
		if e.IsDirty() && !e.quitWarningShown {
			e.quitWarningShown = true
			e.SetStatusMessage("Warning! File has unsaved changes. Press Ctrl-Q or Ctrl-C again to quit.")
			return nil
		}
		return NewQuitCommand(e)
	default:
		return nil
	}
}

// setStatusMessage はステータスメッセージを設定する（非公開メソッド）
func (e *Editor) setStatusMessage(format string, args ...interface{}) {
	if e.config.DebugMode {
		format = "[in Debug] " + format
	}
	// UIコンポーネントのSetMessageメソッドを呼び出す
	e.screen.SetMessage(format, args...)

	// 即座に画面を更新して変更を反映
	e.RefreshScreen()
}

// SetStatusMessage はステータスメッセージを設定する（EditorOperations用の公開メソッド）
func (e *Editor) SetStatusMessage(format string, args ...interface{}) {
	// まずステータスメッセージを直接設定
	e.statusMessage = fmt.Sprintf(format, args...)
	e.statusMessageTime = e.config.StatusMessageDuration

	// イベントとしては発行しない - 無限ループを防ぐ
	e.setStatusMessage(format, args...)
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

func (e *Editor) InsertChar(ch rune) {
	e.logger.Log("edit", fmt.Sprintf("Inserting character: %c", ch))
	pos := e.screen.GetCursor().ToPosition()
	e.buffer.InsertChar(contents.Position{X: pos.X, Y: pos.Y}, ch)
	// カーソルを1つ進める
	e.screen.SetCursorPosition(pos.X+1, pos.Y)
	e.RefreshScreen()
}

func (e *Editor) DeleteChar() {
	e.logger.Log("edit", "Deleting character")
	pos := e.screen.GetCursor().ToPosition()

	if pos.X > 0 {
		// 行の途中での削除
		e.buffer.DeleteChar(contents.Position{X: pos.X, Y: pos.Y})
		e.screen.SetCursorPosition(pos.X-1, pos.Y) // カーソルを1つ左に移動
	} else if pos.Y > 0 {
		// 行頭での削除（前の行との結合）
		prevRow := e.buffer.GetRow(pos.Y - 1)
		if prevRow != nil {
			targetX := prevRow.GetRuneCount() // 前の行の末尾位置
			e.buffer.DeleteChar(contents.Position{X: pos.X, Y: pos.Y})
			e.screen.SetCursorPosition(targetX, pos.Y-1) // 前の行の末尾へ移動
		}
	}

	e.RefreshScreen()
}

func (e *Editor) MoveCursor(movement cursor.Movement) {
	e.logger.Log("cursor", fmt.Sprintf("Moving cursor: %v", movement))
	// Buffer直接操作からUI経由に変更
	e.screen.MoveCursor(movement, e.buffer)
	e.UpdateScroll()
}

func (e *Editor) InsertNewline() {
	e.logger.Log("edit", "Inserting newline")
	cursor := e.screen.GetCursor()
	pos := cursor.ToPosition()
	e.buffer.InsertNewline(contents.Position{X: pos.X, Y: pos.Y})
	// UIに改行処理を通知
	cursor.NewLine()
	e.screen.SetCursor(cursor)
	e.UpdateScroll()
	e.RefreshScreen()
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

func (e *Editor) InsertChars(chars []rune) {
	cursor := e.screen.GetCursor()
	pos := cursor.ToPosition()
	e.buffer.InsertChars(contents.Position{X: pos.X, Y: pos.Y}, chars)
	// カーソルを入力文字数分進める
	e.screen.SetCursorPosition(pos.X+len(chars), pos.Y)
}

// periodicSnapshot は定期的にスナップショットを作成する
func (e *Editor) periodicSnapshot() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if e.buffer != nil && e.buffer.IsDirty() {
				e.stateManager.CreateSnapshot(e.eventManager.GetCurrentEvents())
			}
		case <-e.quit:
			return
		}
	}
}

// saveBufferToTempFile はバッファの内容を一時ファイルに保存する
func (e *Editor) saveBufferToTempFile() {
	baseName := "untitled"
	if filename := e.fileManager.GetFilename(); filename != "" {
		baseName = filename
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
	return e.stateManager.RecoverFromSnapshot(time.Now())
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

// GetEventManager はEventManagerを返す
func (e *Editor) GetEventManager() *events.EventManager {
	return e.eventManager
}

// GetFilename は現在のファイル名を返す
func (e *Editor) GetFilename() string {
	return e.fileManager.GetFilename()
}

// HandleSaveEvent はSaveEventを処理する
func (e *Editor) HandleSaveEvent(event *events.SaveEvent) error {
	return e.fileManager.HandleSaveRequest(event)
}

// HandleQuitEvent はQuitEventを処理する
func (e *Editor) HandleQuitEvent(event *events.QuitEvent) error {
	if e.buffer.IsDirty() && !event.Force {
		event.SaveNeeded = true
		return fmt.Errorf("unsaved changes")
	}
	e.isQuitting = true
	close(e.quit)
	return nil
}

// HandleStatusEvent はStatusEventを処理する
func (e *Editor) HandleStatusEvent(event *events.StatusEvent) error {
	e.SetStatusMessage(event.Message, event.Args...)
	return nil
}

// setupStartupHandlers はエディタの起動時の初期設定を行う
func (e *Editor) setupStartupHandlers() error {
	// 1. 必須コンポーネントの検証
	if err := e.validateComponents(); err != nil {
		return fmt.Errorf("component validation failed: %w", err)
	}

	// 2. リカバリー設定の初期化
	// e.recoveryManager.SetStrategy(events.RollbackToStable)

	// 3. 定期タスクの開始（エディタが終了していない場合のみ）
	if !e.isQuitting {
		go e.periodicSnapshot()
		go e.monitorErrors()
	}

	return nil
}

// validateComponents は必須コンポーネントの存在を検証する
func (e *Editor) validateComponents() error {
	if e.eventManager == nil {
		return fmt.Errorf("event manager is not initialized")
	}
	// if e.recoveryManager == nil {
	// 	return fmt.Errorf("recovery manager is not initialized")
	// }
	if e.stateManager == nil {
		return fmt.Errorf("state manager is not initialized")
	}
	if e.buffer == nil {
		return fmt.Errorf("buffer is not initialized")
	}
	return nil
}
