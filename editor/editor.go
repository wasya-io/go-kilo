package editor

import (
	"os"

	"golang.org/x/sys/unix"
)

// Editor はエディタの状態を管理する構造体
type Editor struct {
	term        *terminalState
	ui          *UI
	quit        chan struct{}
	buffer      *Buffer
	rowOffset   int
	colOffset   int
	fileManager *FileManager
	input       *InputHandler
	config      *Config
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
		config:    LoadConfig(),
	}

	e.fileManager = NewFileManager(e.buffer)
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

// RefreshScreen は画面を更新する
func (e *Editor) RefreshScreen() error {
	// スクロール位置の更新はProcessKeypress内で行うため、ここでは行わない
	return e.ui.RefreshScreen(e.buffer, e.fileManager.GetFilename(), e.rowOffset, e.colOffset)
}

// ProcessKeypress はキー入力を処理する
func (e *Editor) ProcessKeypress() error {
	err := e.input.ProcessKeypress()
	if err != nil {
		return err
	}

	// カーソル位置の移動後に必ずスクロール位置を更新
	e.UpdateScroll()

	// 画面の更新前に、スクロール位置が適切な範囲内にあることを確認
	if e.rowOffset > e.buffer.GetLineCount()-1 {
		e.rowOffset = max(0, e.buffer.GetLineCount()-1)
	}

	return e.RefreshScreen()
}

// max は2つの整数のうち大きい方を返す
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// UpdateScroll はカーソル位置に基づいてスクロール位置を更新する
func (e *Editor) UpdateScroll() {
	// カーソル位置が画面外に出ないようにスクロール位置を調整
	if e.buffer.cursor.Y < e.rowOffset {
		e.rowOffset = e.buffer.cursor.Y
	}

	// 画面下端から2行分の余裕を持たせる（ステータスバーとメッセージバー用）
	screenBottom := e.ui.screenRows - 2
	visibleLines := screenBottom - 1 // 実際に表示可能な行数

	// カーソルが画面下端に近づいた場合のスクロール処理
	if e.buffer.cursor.Y >= e.rowOffset+visibleLines {
		// カーソル位置を中心にスクロール
		e.rowOffset = e.buffer.cursor.Y - visibleLines + 1
	}

	// カーソル行の取得と水平スクロールの処理
	row := e.buffer.getRow(e.buffer.cursor.Y)
	if row == nil {
		return
	}

	// カーソル位置の表示位置を計算
	cursorScreenPos := row.OffsetToScreenPosition(e.buffer.cursor.X)

	// 水平スクロールの調整
	// 左方向のスクロール
	if cursorScreenPos < e.colOffset {
		e.colOffset = cursorScreenPos
	}

	// 右方向のスクロール（画面幅の80%を超えたらスクロール）
	rightMargin := (e.ui.screenCols * 4) / 5
	if cursorScreenPos >= e.colOffset+rightMargin {
		e.colOffset = cursorScreenPos - rightMargin + 1
	}

	// スクロール位置が有効な範囲内に収まるように調整
	if e.rowOffset < 0 {
		e.rowOffset = 0
	}
	if e.colOffset < 0 {
		e.colOffset = 0
	}

	// 最大スクロール位置の制限
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
	close(e.quit)
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

// setStatusMessage はステータスメッセージを設定する
func (e *Editor) setStatusMessage(format string, args ...interface{}) {
	e.ui.SetMessage(format, args...)
}
