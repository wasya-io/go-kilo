package editor

import "golang.org/x/text/width"

// Buffer はテキストバッファ全体を管理する
type Buffer struct {
	rows   []*Row
	cx, cy int
	dirty  bool
}

// NewBuffer は新しいBufferを作成する
func NewBuffer() *Buffer {
	return &Buffer{
		rows: make([]*Row, 0),
		cx:   0,
		cy:   0,
	}
}

// InsertChar は現在のカーソル位置に文字を挿入する
func (b *Buffer) InsertChar(ch rune) {
	if b.cy == len(b.rows) {
		b.rows = append(b.rows, NewRow(""))
	}

	row := b.rows[b.cy]
	row.InsertChar(b.cx, ch)
	b.cx++
	b.dirty = true
}

// DeleteChar はカーソル位置の前の文字を削除する
func (b *Buffer) DeleteChar() {
	if b.cy == len(b.rows) {
		return
	}
	if b.cx == 0 && b.cy == 0 {
		return
	}

	row := b.rows[b.cy]
	if b.cx > 0 {
		row.DeleteChar(b.cx - 1)
		b.cx--
	} else {
		if b.cy > 0 {
			prevRow := b.rows[b.cy-1]
			b.cx = prevRow.GetRuneCount()
			prevRow.Append(row.GetContent())
			b.rows = append(b.rows[:b.cy], b.rows[b.cy+1:]...)
			b.cy--
		}
	}
	b.dirty = true
}

// InsertNewline は現在のカーソル位置で改行を挿入する
func (b *Buffer) InsertNewline() {
	if b.cx == 0 {
		b.rows = append(b.rows[:b.cy], append([]*Row{NewRow("")}, b.rows[b.cy:]...)...)
	} else {
		row := b.rows[b.cy]
		content := row.GetContent()
		runes := []rune(content)
		newRow := NewRow(string(runes[b.cx:]))
		row.chars = string(runes[:b.cx])
		row.updateWidths()
		b.rows = append(b.rows[:b.cy+1], append([]*Row{newRow}, b.rows[b.cy+1:]...)...)
	}
	b.cy++
	b.cx = 0
	b.dirty = true
}

// MoveCursor はカーソルを移動する
func (b *Buffer) MoveCursor(direction CursorMovement) {
	switch direction {
	case CursorUp:
		if b.cy > 0 {
			currentRow := b.rows[b.cy]
			targetScreenPos := currentRow.OffsetToScreenPosition(b.cx)
			b.cy--
			newRow := b.rows[b.cy]
			b.cx = newRow.ScreenPositionToOffset(targetScreenPos)
		}
	case CursorDown:
		if b.cy < len(b.rows)-1 {
			currentRow := b.rows[b.cy]
			targetScreenPos := currentRow.OffsetToScreenPosition(b.cx)
			b.cy++
			newRow := b.rows[b.cy]
			b.cx = newRow.ScreenPositionToOffset(targetScreenPos)
		}
	case CursorRight:
		if b.cy < len(b.rows) {
			runes := []rune(b.rows[b.cy].GetContent())
			if b.cx < len(runes) {
				b.cx++
			} else if b.cy < len(b.rows)-1 {
				b.cy++
				b.cx = 0
			}
		}
	case CursorLeft:
		if b.cx > 0 {
			b.cx--
		} else if b.cy > 0 {
			b.cy--
			runes := []rune(b.rows[b.cy].GetContent())
			b.cx = len(runes)
		}
	}
}

// GetCursor はカーソル位置を返す
func (b *Buffer) GetCursor() (int, int) {
	return b.cx, b.cy
}

// SetCursor はカーソル位置を設定する
func (b *Buffer) SetCursor(x, y int) {
	if y >= 0 && y < len(b.rows) {
		b.cy = y
		runes := []rune(b.rows[y].GetContent())
		if x >= 0 {
			if x > len(runes) {
				x = len(runes)
			}
			b.cx = x
		}
	}
}

// GetContent は指定された行の内容を返す
func (b *Buffer) GetContent(lineNum int) string {
	if lineNum >= 0 && lineNum < len(b.rows) {
		return b.rows[lineNum].GetContent()
	}
	return ""
}

// GetCharAtCursorPosition は指定された位置の文字を返す
func (b *Buffer) GetCharAtCursorPosition(cx, cy int) string {
	if content := b.GetContent(cy); content != "" {
		runes := []rune(content)
		if cx < len(runes) {
			return string(runes[cx])
		}
	}
	return ""
}

// GetCharAtCursor は現在のカーソル位置の文字を返す
func (b *Buffer) GetCharAtCursor() string {
	cx, cy := b.GetCursor()
	return b.GetCharAtCursorPosition(cx, cy)
}

// GetLineCount は行数を返す
func (b *Buffer) GetLineCount() int {
	return len(b.rows)
}

// IsDirty は未保存の変更があるかどうかを返す
func (b *Buffer) IsDirty() bool {
	return b.dirty
}

// SetDirty は未保存の変更状態を設定する
func (b *Buffer) SetDirty(dirty bool) {
	b.dirty = dirty
}

// LoadContent はバッファの内容を設定する
func (b *Buffer) LoadContent(lines []string) {
	b.rows = make([]*Row, len(lines))
	for i, line := range lines {
		b.rows[i] = NewRow(line)
	}
	b.dirty = false
}

// GetContent は全行の内容を文字列のスライスとして返す
func (b *Buffer) GetAllContent() []string {
	content := make([]string, len(b.rows))
	for i, row := range b.rows {
		content[i] = row.GetContent()
	}
	return content
}

// Row は1行のテキストデータと関連情報を保持する
type Row struct {
	chars     string // 実際の文字列データ
	widths    []int  // 各文字の表示幅
	positions []int  // 各文字の表示位置（累積幅）
}

// NewRow は新しいRow構造体を作成する
func NewRow(chars string) *Row {
	r := &Row{
		chars: chars,
	}
	r.updateWidths()
	return r
}

// updateWidths は行の文字幅情報を更新する
func (r *Row) updateWidths() {
	runes := []rune(r.chars)
	r.widths = make([]int, len(runes))
	r.positions = make([]int, len(runes)+1)
	pos := 0

	for i, ch := range runes {
		w := getCharWidth(ch)
		r.widths[i] = w
		r.positions[i] = pos
		pos += w
	}
	r.positions[len(runes)] = pos // 最後の位置も記録
}

// InsertChar は指定位置に文字を挿入する
func (r *Row) InsertChar(at int, ch rune) {
	runes := []rune(r.chars)
	if at > len(runes) {
		at = len(runes)
	}

	runes = append(runes[:at], append([]rune{ch}, runes[at:]...)...)
	r.chars = string(runes)
	r.updateWidths()
}

// DeleteChar は指定位置の文字を削除する
func (r *Row) DeleteChar(at int) {
	runes := []rune(r.chars)
	if at >= len(runes) {
		return
	}

	r.chars = string(append(runes[:at], runes[at+1:]...))
	r.updateWidths()
}

// ScreenPositionToOffset は画面上の位置から文字列中のオフセットを取得する
func (r *Row) ScreenPositionToOffset(screenPos int) int {
	for i, pos := range r.positions {
		if pos > screenPos {
			return i - 1
		}
	}
	return len([]rune(r.chars))
}

// OffsetToScreenPosition は文字列中のオフセットから画面上の位置を取得する
func (r *Row) OffsetToScreenPosition(offset int) int {
	runes := []rune(r.chars)
	if offset >= len(runes) {
		return r.positions[len(runes)]
	}
	return r.positions[offset]
}

// GetContent は行の内容を文字列として返す
func (r *Row) GetContent() string {
	return r.chars
}

// GetRuneCount は行の文字数を返す
func (r *Row) GetRuneCount() int {
	return len([]rune(r.chars))
}

// Append は行の末尾に文字列を追加する
func (r *Row) Append(s string) {
	r.chars += s
	r.updateWidths()
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
