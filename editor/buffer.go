package editor

import "golang.org/x/text/width"

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
