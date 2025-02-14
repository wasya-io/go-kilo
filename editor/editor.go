package editor

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// Editor はエディタの状態を管理する構造体
type Editor struct {
	term        *terminalState
	screenRows  int
	screenCols  int
	quit        chan struct{}
	rows        []string
	cx, cy      int
	filename    string    // 編集中のファイル名
	dirty       bool      // 未保存の変更があるかどうか
	message     string    // ステータスメッセージ
	messageTime time.Time // メッセージの表示時間
}

// New は新しいEditorインスタンスを作成する
func New() (*Editor, error) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return nil, err
	}

	e := &Editor{
		screenRows: int(ws.Row),
		screenCols: int(ws.Col),
		quit:       make(chan struct{}),
		rows:       make([]string, 0),
		cx:         0,
		cy:         0,
		dirty:      false,
	}

	// テスト用のダミーテキストを追加
	e.rows = append(e.rows, "Hello, Go-Kilo editor!")
	e.rows = append(e.rows, "Use arrow keys to move cursor.")
	e.rows = append(e.rows, "Press Ctrl-Q or Ctrl-C to quit.")

	// Rawモードを有効化
	term, err := enableRawMode()
	if err != nil {
		return nil, err
	}
	e.term = term

	return e, nil
}

// Cleanup は終了時の後処理を行う
func (e *Editor) Cleanup() {
	e.term.disableRawMode()
	fmt.Print("\x1b[2J")
	fmt.Print("\x1b[H")
}

// RefreshScreen は画面を更新する
func (e *Editor) RefreshScreen() error {
	var b strings.Builder

	b.WriteString("\x1b[?25l")
	b.WriteString("\x1b[H")

	// テキストエリアの描画
	for y := 0; y < e.screenRows-2; y++ {
		if y < len(e.rows) {
			if len(e.rows[y]) > e.screenCols {
				b.WriteString(e.rows[y][:e.screenCols])
			} else {
				b.WriteString(e.rows[y])
			}
		} else {
			b.WriteString("~")
		}

		b.WriteString("\x1b[K")
		b.WriteString("\r\n")
	}

	// ステータスバーの描画
	b.WriteString("\x1b[7m") // 反転表示
	status := ""
	if e.filename == "" {
		status = "[No Name]"
	} else {
		status = e.filename
	}
	if e.dirty {
		status += " [+]"
	}
	rstatus := fmt.Sprintf("%d/%d", e.cy+1, len(e.rows))
	if len(status) > e.screenCols {
		status = status[:e.screenCols]
	}
	b.WriteString(status)
	for i := len(status); i < e.screenCols-len(rstatus); i++ {
		b.WriteString(" ")
	}
	b.WriteString(rstatus)
	b.WriteString("\x1b[m") // 反転表示解除
	b.WriteString("\r\n")

	// メッセージ行の描画
	b.WriteString("\x1b[K")
	if time.Since(e.messageTime) < 5*time.Second {
		b.WriteString(e.message)
	}

	// カーソル位置の設定
	fmt.Fprintf(&b, "\x1b[%d;%dH", e.cy+1, e.cx+1)
	b.WriteString("\x1b[?25h")

	_, err := fmt.Print(b.String())
	return err
}

// ProcessKeypress はキー入力を処理する
func (e *Editor) ProcessKeypress() error {
	buf := make([]byte, 1)
	_, err := os.Stdin.Read(buf)
	if err != nil {
		return err
	}

	switch buf[0] {
	case 'q' & 0x1f, 'c' & 0x1f: // Ctrl-Q または Ctrl-C
		if e.dirty {
			e.setStatusMessage("Warning! File has unsaved changes. Press Ctrl-Q or Ctrl-C again to quit.")
			e.dirty = false
			return nil
		}
		close(e.quit)
		return nil
	case 's' & 0x1f: // Ctrl-S
		if err := e.SaveFile(); err != nil {
			e.setStatusMessage("Can't save! I/O error: %s", err)
		}
		return nil
	case '\x1b':
		if err := e.readEscapeSequence(); err != nil {
			return err
		}
	case '\r':
		e.insertNewline()
	case 127:
		e.deleteChar()
	default:
		if !iscntrl(buf[0]) {
			e.insertChar(rune(buf[0]))
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

// readEscapeSequence は矢印キーなどのエスケープシーケンスを読み取る
func (e *Editor) readEscapeSequence() error {
	buf := make([]byte, 2)
	_, err := os.Stdin.Read(buf)
	if err != nil {
		return err
	}

	if buf[0] == '[' {
		switch buf[1] {
		case 'A': // 上矢印
			if e.cy > 0 {
				e.cy--
			}
		case 'B': // 下矢印
			if e.cy < len(e.rows)-1 {
				e.cy++
			}
		case 'C': // 右矢印
			if e.cx < e.screenCols-1 {
				if e.cy < len(e.rows) && e.cx < len(e.rows[e.cy]) {
					e.cx++
				}
			}
		case 'D': // 左矢印
			if e.cx > 0 {
				e.cx--
			}
		}
	}
	return nil
}

// iscntrl は制御文字かどうかを判定する
func iscntrl(b byte) bool {
	return b < 32 || b == 127
}

// insertChar は現在のカーソル位置に文字を挿入する
func (e *Editor) insertChar(ch rune) {
	if e.cy == len(e.rows) {
		e.rows = append(e.rows, "")
	}

	row := []rune(e.rows[e.cy])
	if e.cx > len(row) {
		e.cx = len(row)
	}

	newRow := make([]rune, 0, len(row)+1)
	newRow = append(newRow, row[:e.cx]...)
	newRow = append(newRow, ch)
	newRow = append(newRow, row[e.cx:]...)

	e.rows[e.cy] = string(newRow)
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

	if e.cx > 0 {
		row := []rune(e.rows[e.cy])
		newRow := make([]rune, 0, len(row)-1)
		newRow = append(newRow, row[:e.cx-1]...)
		newRow = append(newRow, row[e.cx:]...)
		e.rows[e.cy] = string(newRow)
		e.cx--
	} else {
		if e.cy > 0 {
			e.cx = len(e.rows[e.cy-1])
			e.rows[e.cy-1] += e.rows[e.cy]
			e.rows = append(e.rows[:e.cy], e.rows[e.cy+1:]...)
			e.cy--
		}
	}
	e.dirty = true
}

// insertNewline は現在のカーソル位置で改行を挿入する
func (e *Editor) insertNewline() {
	if e.cx == 0 {
		e.rows = append(e.rows[:e.cy], append([]string{""}, e.rows[e.cy:]...)...)
	} else {
		row := e.rows[e.cy]
		e.rows = append(e.rows[:e.cy], append([]string{
			row[:e.cx],
			row[e.cx:],
		}, e.rows[e.cy+1:]...)...)
	}
	e.cy++
	e.cx = 0
	e.dirty = true
}

// OpenFile は指定されたファイルを読み込む
func (e *Editor) OpenFile(filename string) error {
	e.filename = filename

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// 新規ファイルの場合は空のバッファを用意
			e.rows = make([]string, 0)
			e.setStatusMessage("New file")
			return nil
		}
		return err
	}
	defer file.Close()

	// ファイル全体を読み込んでから行に分割
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// 改行で分割して行を取得
	e.rows = strings.Split(strings.TrimRight(string(content), "\n"), "\n")
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

	content := strings.Join(e.rows, "\n")
	err := os.WriteFile(e.filename, []byte(content), 0644)
	if err != nil {
		return err
	}

	e.dirty = false
	e.setStatusMessage("File saved")
	return nil
}

// setStatusMessage はステータスメッセージを設定する
func (e *Editor) setStatusMessage(format string, args ...interface{}) {
	e.message = fmt.Sprintf(format, args...)
	e.messageTime = time.Now()
}
