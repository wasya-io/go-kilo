package editor

import (
	"os"

	"golang.org/x/sys/unix"
)

// Editor はエディタの状態を管理する構造体
type Editor struct {
	term      *terminalState
	ui        *UI
	quit      chan struct{}
	buffer    *Buffer
	rowOffset int
	colOffset int
	filename  string
	storage   Storage
	input     *InputHandler
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

	e := &Editor{
		ui:        NewUI(screenRows, screenCols),
		quit:      make(chan struct{}),
		buffer:    NewBuffer(),
		rowOffset: 0,
		colOffset: 0,
		storage:   NewFileStorage(),
	}

	e.input = NewInputHandler(e)

	if !testMode {
		// テスト以外の場合のみデフォルトテキストを追加
		defaultContent := []string{
			"Hello, Go-Kilo editor!",
			"Use arrow keys to move cursor.",
			"Press Ctrl-Q or Ctrl-C to quit.",
		}
		e.buffer.LoadContent(defaultContent)

		// Rawモードを有効化
		term, err := enableRawMode()
		if err != nil {
			return nil, err
		}
		e.term = term
	}

	return e, nil
}

// Cleanup は終了時の後処理を行う
func (e *Editor) Cleanup() {
	if e.term != nil {
		e.term.disableRawMode()
	}
	os.Stdout.WriteString(e.ui.clearScreen())
	os.Stdout.WriteString(e.ui.moveCursorToHome())
}

// scroll は必要に応じてスクロール位置を更新する
func (e *Editor) scroll() {
	cx, cy := e.buffer.GetCursor()

	// 垂直スクロール
	if cy < e.rowOffset {
		e.rowOffset = cy
	}
	if cy >= e.rowOffset+e.ui.screenRows-2 {
		e.rowOffset = cy - (e.ui.screenRows - 3)
	}

	// 水平スクロール
	screenX := 0
	if cy < e.buffer.GetLineCount() {
		row := e.buffer.GetContent(cy)
		if len(row) > 0 {
			screenX = len([]rune(row[:cx]))
		}
	}

	if screenX < e.colOffset {
		e.colOffset = screenX
	}
	if screenX >= e.colOffset+e.ui.screenCols {
		e.colOffset = screenX - e.ui.screenCols + 1
	}
}

// RefreshScreen は画面を更新する
func (e *Editor) RefreshScreen() error {
	e.scroll()
	lines := make([]*Row, e.buffer.GetLineCount())
	for i := 0; i < e.buffer.GetLineCount(); i++ {
		content := e.buffer.GetContent(i)
		lines[i] = NewRow(content)
	}
	cx, cy := e.buffer.GetCursor()
	output := e.ui.RenderScreen(lines, e.filename, e.buffer.IsDirty(), cx, cy, e.rowOffset, e.colOffset)
	_, err := os.Stdout.WriteString(output)
	return err
}

// ProcessKeypress はキー入力を処理する
func (e *Editor) ProcessKeypress() error {
	return e.input.ProcessKeypress()
}

// SetKeyReader はキー入力読み取りインターフェースを設定する
func (e *Editor) SetKeyReader(reader KeyReader) {
	e.input.SetKeyReader(reader)
}

// QuitChan は終了シグナルを監視するためのチャネルを返す
func (e *Editor) QuitChan() <-chan struct{} {
	return e.quit
}

// Quit はエディタを終了する
func (e *Editor) Quit() {
	close(e.quit)
}

// OpenFile は指定されたファイルを読み込む
func (e *Editor) OpenFile(filename string) error {
	e.filename = filename

	lines, err := e.storage.Load(filename)
	if err != nil {
		return err
	}

	e.buffer.LoadContent(lines)
	e.setStatusMessage("File loaded")
	return nil
}

// SaveFile は現在の内容をファイルに保存する
func (e *Editor) SaveFile() error {
	if e.filename == "" {
		e.setStatusMessage("No filename")
		return nil
	}

	content := e.buffer.GetAllContent()
	err := e.storage.Save(e.filename, content)
	if err != nil {
		return err
	}

	e.buffer.SetDirty(false)
	e.setStatusMessage("File saved")
	return nil
}

// setStatusMessage はステータスメッセージを設定する
func (e *Editor) setStatusMessage(format string, args ...interface{}) {
	e.ui.SetMessage(format, args...)
}

// GetCharAtCursor は現在のカーソル位置の文字を返す
func (e *Editor) GetCharAtCursor() string {
	cx, cy := e.buffer.GetCursor()
	if content := e.buffer.GetContent(cy); content != "" {
		runes := []rune(content)
		if cx < len(runes) {
			return string(runes[cx])
		}
	}
	return ""
}

// GetContent は指定された行の内容を返す
func (e *Editor) GetContent(lineNum int) string {
	return e.buffer.GetContent(lineNum)
}

// GetLineCount は行数を返す
func (e *Editor) GetLineCount() int {
	return e.buffer.GetLineCount()
}
