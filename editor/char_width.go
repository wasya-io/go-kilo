package editor

import "golang.org/x/text/width"

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
