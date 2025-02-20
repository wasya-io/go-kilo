package editor

import (
	"os"

	"github.com/wasya-io/go-kilo/editor/events"
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

	e := &Editor{
		ui:           NewUI(screenRows, screenCols, eventManager), // eventManagerを追加
		quit:         make(chan struct{}),
		buffer:       NewBuffer(eventManager), // eventManagerを引数として渡す
		rowOffset:    0,
		colOffset:    0,
		config:       LoadConfig(),
		eventManager: eventManager,
		isQuitting:   false,
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
	}

	return e, nil
}

// setupEventHandlers はイベントハンドラを設定する
func (e *Editor) setupEventHandlers() {
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
	if e.term != nil {
		e.term.disableRawMode()
	}
	os.Stdout.WriteString(e.ui.clearScreen())
	os.Stdout.WriteString(e.ui.moveCursorToHome())
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
		return err
	}

	// 画面更新を必ず行う（コマンドの有無に関わらず）
	defer e.RefreshScreen()

	if command != nil {
		// コマンドを実行
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
	if !e.isQuitting {
		e.isQuitting = true
		close(e.quit)
	}
}

// OpenFile は指定されたファイルを読み込む
func (e *Editor) OpenFile(filename string) error {
	if err := e.fileManager.OpenFile(filename); err != nil {
		return err
	}
	e.setStatusMessage("File loaded")
	return nil
}

// SaveFile は現在の内容をファイルに保存する
func (e *Editor) SaveFile() error {
	if err := e.fileManager.SaveFile(); err != nil {
		if err == ErrNoFilename {
			e.setStatusMessage("No filename")
			return nil
		}
		return err
	}
	e.setStatusMessage("File saved")
	return nil
}

// setStatusMessage はステータスメッセージを設定する（非公開メソッド）
func (e *Editor) setStatusMessage(format string, args ...interface{}) {
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
	e.buffer.InsertChar(ch)
	e.RefreshScreen()
}

func (e *Editor) DeleteChar() {
	e.buffer.DeleteChar()
	e.RefreshScreen()
}

func (e *Editor) MoveCursor(movement CursorMovement) {
	// バッファのカーソル移動を実行
	e.buffer.MoveCursor(movement)
	// スクロール位置の更新
	e.UpdateScroll()
	// イベント発行は buffer.MoveCursor 内で行われるため、ここでは不要
}

func (e *Editor) InsertNewline() {
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
	return e.buffer.GetContent(lineNum)
}

func (e *Editor) InsertChars(chars []rune) {
	e.buffer.InsertChars(chars)
}
