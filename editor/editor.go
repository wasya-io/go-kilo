package editor

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/width"

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
	rowOffset   int       // スクロール位置（垂直）
	colOffset   int       // スクロール位置（水平）
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
		rowOffset:  0,
		colOffset:  0,
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

// 文字の表示幅を取得する
func getCharWidth(ch rune) int {
	p := width.LookupRune(ch)
	switch p.Kind() {
	case width.EastAsianFullwidth, width.EastAsianWide:
		return 2
	default:
		return 1
	}
}

// 文字列の表示幅を計算する
func getStringWidth(s string) int {
	width := 0
	for _, ch := range s {
		width += getCharWidth(ch)
	}
	return width
}

// スクリーン上の位置から文字列内のバイト位置を取得
func getOffsetFromScreenPos(s string, screenPos int) int {
	currentWidth := 0
	currentOffset := 0

	for currentOffset < len(s) {
		r, size := utf8.DecodeRuneInString(s[currentOffset:])
		if r == utf8.RuneError {
			return currentOffset
		}

		charWidth := getCharWidth(r)
		if currentWidth+charWidth > screenPos {
			break
		}

		currentWidth += charWidth
		currentOffset += size
	}

	return currentOffset
}

// 文字列内のバイト位置からスクリーン上の位置を取得
func getScreenPosFromOffset(s string, offset int) int {
	width := 0
	for i := 0; i < offset; {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError {
			break
		}
		width += getCharWidth(r)
		i += size
	}
	return width
}

// scroll は必要に応じてスクロール位置を更新する
func (e *Editor) scroll() {
	// 垂直スクロール
	if e.cy < e.rowOffset {
		e.rowOffset = e.cy
	}
	if e.cy >= e.rowOffset+e.screenRows-2 {
		e.rowOffset = e.cy - (e.screenRows - 3)
	}

	// 水平スクロール
	screenX := 0
	if e.cy < len(e.rows) {
		screenX = getScreenPosFromOffset(e.rows[e.cy], e.cx)
	}

	if screenX < e.colOffset {
		e.colOffset = screenX
	}
	if screenX >= e.colOffset+e.screenCols {
		e.colOffset = screenX - e.screenCols + 1
	}
}

// RefreshScreen は画面を更新する
func (e *Editor) RefreshScreen() error {
	e.scroll()

	var b strings.Builder

	b.WriteString("\x1b[?25l")
	b.WriteString("\x1b[H")

	// テキストエリアの描画
	for y := 0; y < e.screenRows-2; y++ {
		filerow := y + e.rowOffset
		if filerow < len(e.rows) {
			line := e.rows[filerow]
			currentWidth := -e.colOffset
			currentOffset := 0

			// スクロール位置まで読み飛ばし
			for currentOffset < len(line) && currentWidth < 0 {
				r, size := utf8.DecodeRuneInString(line[currentOffset:])
				if r == utf8.RuneError {
					break
				}
				currentWidth += getCharWidth(r)
				currentOffset += size
			}

			// 表示範囲の文字を描画
			for currentOffset < len(line) {
				r, size := utf8.DecodeRuneInString(line[currentOffset:])
				if r == utf8.RuneError {
					break
				}

				charWidth := getCharWidth(r)
				if currentWidth+charWidth > e.screenCols {
					break
				}

				b.WriteString(line[currentOffset : currentOffset+size])
				currentWidth += charWidth
				currentOffset += size
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

	// カーソル位置の設定（スクロール位置を考慮）
	screenX := 1
	if e.cy < len(e.rows) {
		screenX = getScreenPosFromOffset(e.rows[e.cy], e.cx) - e.colOffset + 1
	}
	filerow := e.cy - e.rowOffset + 1
	fmt.Fprintf(&b, "\x1b[%d;%dH", filerow, screenX)
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
			if e.cy < len(e.rows) {
				// 現在行の末尾までカーソル移動可能
				if e.cx < len(e.rows[e.cy]) {
					e.cx++
				}
			}
		case 'D': // 左矢印
			if e.cx > 0 {
				e.cx--
			}
		}

		// 行移動時のカーソル位置調整
		if e.cy < len(e.rows) {
			rowLen := len(e.rows[e.cy])
			if e.cx > rowLen {
				e.cx = rowLen
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

	row := e.rows[e.cy]
	if e.cx > len(row) {
		e.cx = len(row)
	}

	// 現在のカーソル位置までの文字列と残りの文字列を結合
	e.rows[e.cy] = row[:e.cx] + string(ch) + row[e.cx:]
	e.cx += utf8.RuneLen(ch)
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
		// カーソル位置の直前の文字のサイズを取得
		_, size := utf8.DecodeLastRuneInString(row[:e.cx])
		e.rows[e.cy] = row[:e.cx-size] + row[e.cx:]
		e.cx -= size
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
