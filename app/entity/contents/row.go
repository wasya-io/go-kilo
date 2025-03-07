package contents

import "golang.org/x/text/width"

// Row は1行のテキストデータと関連情報を保持する
type Row struct {
	chars      string
	runeSlice  []rune
	widths     []int
	positions  []int
	totalWidth int
}

// NewRow は新しいRow構造体を作成する
func NewRow(chars string) *Row {
	if chars == "" {
		return &Row{
			chars:      "",
			runeSlice:  []rune{},
			widths:     []int{},
			positions:  []int{0}, // 空の行でも最初の位置（0）は必要
			totalWidth: 0,
		}
	}

	runeSlice := []rune(chars)
	r := &Row{
		chars:      chars,
		runeSlice:  runeSlice,
		widths:     make([]int, len(runeSlice)),
		positions:  make([]int, len(runeSlice)+1),
		totalWidth: 0,
	}
	r.updateWidths()
	return r
}

// GetContent は行の内容を文字列として返す
func (r *Row) GetContent() string {
	return r.chars
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
	if at < 0 || at >= len(r.runeSlice) {
		return
	}
	r.runeSlice = append(r.runeSlice[:at], r.runeSlice[at+1:]...)
	r.chars = string(r.runeSlice)
	r.updateWidths()
}

// ScreenPositionToOffset は画面上の位置から文字列中のオフセットを取得する
func (r *Row) ScreenPositionToOffset(screenPos int) int {
	if len(r.positions) == 0 {
		return 0
	}

	// 画面位置が行の最後を超える場合は最後の文字の位置を返す
	if screenPos >= r.totalWidth {
		return len(r.runeSlice)
	}
	if screenPos < 0 {
		return 0
	}

	// 画面位置に最も近い文字位置を探す
	for i := 0; i < len(r.runeSlice); i++ {
		start := r.positions[i]
		end := r.positions[i] + r.widths[i]

		// 画面位置が現在の文字の範囲内にある場合
		if screenPos >= start && screenPos < end {
			return i
		}
	}

	// 見つからない場合は最後の文字の次の位置を返す
	return len(r.runeSlice)
}

// OffsetToScreenPosition は文字列中のオフセットから画面上の位置を取得する
func (r *Row) OffsetToScreenPosition(offset int) int {
	if offset < 0 {
		return 0
	}
	if offset >= len(r.runeSlice) {
		return r.totalWidth
	}
	return r.positions[offset]
}

// GetRuneCount は行の文字数を返す
func (r *Row) GetRuneCount() int {
	return len(r.runeSlice)
}

func (r *Row) GetRunes() []rune {
	return r.runeSlice
}

// GetRuneAt は指定された位置のルーンを返す
func (r *Row) GetRuneAt(offset int) (rune, bool) {
	if offset < 0 || offset >= len(r.runeSlice) {
		return 0, false
	}
	return r.runeSlice[offset], true
}

// GetRuneWidth は指定された位置の文字の表示幅を返す
func (r *Row) GetRuneWidth(offset int) int {
	if offset < 0 || offset >= len(r.widths) {
		return 0
	}
	return r.widths[offset]
}

// updateWidths は行の文字幅情報を更新する
func (r *Row) updateWidths() {
	// runeSliceがnilの場合は初期化
	if r.runeSlice == nil {
		r.runeSlice = []rune(r.chars)
	}

	// widthsとpositionsの長さを確認し、必要に応じて再割り当て
	if len(r.widths) != len(r.runeSlice) {
		r.widths = make([]int, len(r.runeSlice))
	}
	if len(r.positions) != len(r.runeSlice)+1 {
		r.positions = make([]int, len(r.runeSlice)+1)
	}

	r.totalWidth = 0
	for i, ch := range r.runeSlice {
		w := getCharWidth(ch)
		r.widths[i] = w
		r.positions[i] = r.totalWidth
		r.totalWidth += w
	}
	r.positions[len(r.runeSlice)] = r.totalWidth

	// 一貫性チェック
	if len(r.runeSlice) != len(r.widths) || len(r.positions) != len(r.runeSlice)+1 {
		// 長さの不整合を修正
		r.widths = make([]int, len(r.runeSlice))
		r.positions = make([]int, len(r.runeSlice)+1)
		r.updateWidths() // 再帰呼び出しは1回限り
	}
}

// getCharWidth は文字の表示幅を返す
func getCharWidth(ch rune) int {
	p := width.LookupRune(ch)
	switch p.Kind() {
	case width.EastAsianFullwidth, width.EastAsianWide:
		return 2
	default:
		return 1
	}
}
