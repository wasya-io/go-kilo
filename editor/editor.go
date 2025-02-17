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
	rows      []*Row
	cx, cy    int
	rowOffset int
	colOffset int
	filename  string
	dirty     bool
	storage   Storage
	keyReader KeyReader
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
		rows:      make([]*Row, 0),
		cx:        0,
		cy:        0,
		rowOffset: 0,
		colOffset: 0,
		dirty:     false,
		storage:   NewFileStorage(),
		keyReader: NewStandardKeyReader(),
	}

	if !testMode {
		// テスト以外の場合のみデフォルトテキストを追加
		e.rows = append(e.rows, NewRow("Hello, Go-Kilo editor!"))
		e.rows = append(e.rows, NewRow("Use arrow keys to move cursor."))
		e.rows = append(e.rows, NewRow("Press Ctrl-Q or Ctrl-C to quit."))

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
	// 垂直スクロール
	if e.cy < e.rowOffset {
		e.rowOffset = e.cy
	}
	if e.cy >= e.rowOffset+e.ui.screenRows-2 {
		e.rowOffset = e.cy - (e.ui.screenRows - 3)
	}

	// 水平スクロール
	screenX := 0
	if e.cy < len(e.rows) {
		screenX = e.rows[e.cy].OffsetToScreenPosition(e.cx)
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
	output := e.ui.RenderScreen(e.rows, e.filename, e.dirty, e.cx, e.cy, e.rowOffset, e.colOffset)
	_, err := os.Stdout.WriteString(output)
	return err
}

// ProcessKeypress はキー入力を処理する
func (e *Editor) ProcessKeypress() error {
	event, err := e.keyReader.ReadKey()
	if err != nil {
		return err
	}

	switch event.Type {
	case KeyEventSpecial:
		return e.handleSpecialKey(event.Key)
	case KeyEventControl:
		return e.handleControlKey(event.Key)
	case KeyEventChar:
		e.insertChar(event.Rune)
	}

	return nil
}

// handleSpecialKey は特殊キーの処理を行う
func (e *Editor) handleSpecialKey(key Key) error {
	switch key {
	case KeyArrowUp:
		if e.cy > 0 {
			currentRow := e.rows[e.cy]
			targetScreenPos := currentRow.OffsetToScreenPosition(e.cx)
			e.cy--
			newRow := e.rows[e.cy]
			e.cx = newRow.ScreenPositionToOffset(targetScreenPos)
		}
	case KeyArrowDown:
		if e.cy < len(e.rows)-1 {
			currentRow := e.rows[e.cy]
			targetScreenPos := currentRow.OffsetToScreenPosition(e.cx)
			e.cy++
			newRow := e.rows[e.cy]
			e.cx = newRow.ScreenPositionToOffset(targetScreenPos)
		}
	case KeyArrowRight:
		if e.cy < len(e.rows) {
			runes := []rune(e.rows[e.cy].GetContent())
			if e.cx < len(runes) {
				e.cx++
			} else if e.cy < len(e.rows)-1 {
				e.cy++
				e.cx = 0
			}
		}
	case KeyArrowLeft:
		if e.cx > 0 {
			e.cx--
		} else if e.cy > 0 {
			e.cy--
			runes := []rune(e.rows[e.cy].GetContent())
			e.cx = len(runes)
		}
	case KeyBackspace:
		e.deleteChar()
	case KeyEnter:
		e.insertNewline()
	}
	return nil
}

// handleControlKey は制御キーの処理を行う
func (e *Editor) handleControlKey(key Key) error {
	switch key {
	case KeyCtrlQ, KeyCtrlC:
		if e.dirty {
			e.setStatusMessage("Warning! File has unsaved changes. Press Ctrl-Q or Ctrl-C again to quit.")
			e.dirty = false
			return nil
		}
		close(e.quit)
	case KeyCtrlS:
		if err := e.SaveFile(); err != nil {
			e.setStatusMessage("Can't save! I/O error: %s", err)
		}
	}
	return nil
}

// QuitChan は終了シグナルを監視するためのチャネルを返す
func (e *Editor) QuitChan() <-chan struct{} {
	return e.quit
}

// Quit はエディタを終了する
func (e *Editor) Quit() {
	close(e.quit)
}

// insertChar は現在のカーソル位置に文字を挿入する
func (e *Editor) insertChar(ch rune) {
	if e.cy == len(e.rows) {
		e.rows = append(e.rows, NewRow(""))
	}

	row := e.rows[e.cy]
	row.InsertChar(e.cx, ch)
	e.cx++
	e.dirty = true
}

// deleteChar はカーソル位置の前の文字を削除する
func (e *Editor) deleteChar() {
	if e.cy == len(e.rows) {
		return
	}
	if e.cx == 0 && e.cy == 0 {
		return
	}

	row := e.rows[e.cy]
	if e.cx > 0 {
		row.DeleteChar(e.cx - 1)
		e.cx--
	} else {
		if e.cy > 0 {
			prevRow := e.rows[e.cy-1]
			e.cx = prevRow.GetRuneCount()
			prevRow.Append(row.GetContent())
			e.rows = append(e.rows[:e.cy], e.rows[e.cy+1:]...)
			e.cy--
		}
	}
	e.dirty = true
}

// insertNewline は現在のカーソル位置で改行を挿入する
func (e *Editor) insertNewline() {
	if e.cx == 0 {
		e.rows = append(e.rows[:e.cy], append([]*Row{NewRow("")}, e.rows[e.cy:]...)...)
	} else {
		row := e.rows[e.cy]
		content := row.GetContent()
		runes := []rune(content)
		newRow := NewRow(string(runes[e.cx:]))
		row.chars = string(runes[:e.cx])
		row.updateWidths()
		e.rows = append(e.rows[:e.cy+1], append([]*Row{newRow}, e.rows[e.cy+1:]...)...)
	}
	e.cy++
	e.cx = 0
	e.dirty = true
}

// OpenFile は指定されたファイルを読み込む
func (e *Editor) OpenFile(filename string) error {
	e.filename = filename

	lines, err := e.storage.Load(filename)
	if err != nil {
		return err
	}

	e.rows = make([]*Row, len(lines))
	for i, line := range lines {
		e.rows[i] = NewRow(line)
	}

	e.dirty = false
	e.setStatusMessage("File loaded")
	return nil
}

// SaveFile は現在の内容をファイルに保存する
func (e *Editor) SaveFile() error {
	if e.filename == "" {
		e.setStatusMessage("No filename")
		return nil
	}

	content := make([]string, len(e.rows))
	for i, row := range e.rows {
		content[i] = row.GetContent()
	}

	err := e.storage.Save(e.filename, content)
	if err != nil {
		return err
	}

	e.dirty = false
	e.setStatusMessage("File saved")
	return nil
}

// setStatusMessage はステータスメッセージを設定する
func (e *Editor) setStatusMessage(format string, args ...interface{}) {
	e.ui.SetMessage(format, args...)
}

// SetKeyReader はキー入力読み取りインターフェースを設定する
func (e *Editor) SetKeyReader(reader KeyReader) {
	e.keyReader = reader
}

// GetCharAtCursor は現在のカーソル位置の文字を返す
func (e *Editor) GetCharAtCursor() string {
	if e.cy >= len(e.rows) {
		return ""
	}
	row := e.rows[e.cy]
	runes := []rune(row.GetContent())
	if e.cx >= len(runes) {
		return ""
	}
	return string(runes[e.cx])
}

// GetContent は指定された行の内容を返す
func (e *Editor) GetContent(lineNum int) string {
	if lineNum >= 0 && lineNum < len(e.rows) {
		return e.rows[lineNum].GetContent()
	}
	return ""
}

// GetLineCount は行数を返す
func (e *Editor) GetLineCount() int {
	return len(e.rows)
}
