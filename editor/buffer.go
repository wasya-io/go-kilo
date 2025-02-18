package editor

// Buffer はテキストバッファを管理する構造体
type Buffer struct {
	lines    []string
	cursor   Cursor
	isDirty  bool
	rowCache map[int]*Row // 行ごとのキャッシュを追加
}

// Cursor はカーソル位置を管理する構造体
type Cursor struct {
	X, Y int
}

// NewBuffer は新しいBufferインスタンスを作成する
func NewBuffer() *Buffer {
	return &Buffer{
		lines:    make([]string, 0),
		cursor:   Cursor{X: 0, Y: 0},
		isDirty:  false,
		rowCache: make(map[int]*Row),
	}
}

// LoadContent はバッファに内容をロードする
func (b *Buffer) LoadContent(lines []string) {
	b.lines = lines
	b.isDirty = false
	b.cursor = Cursor{X: 0, Y: 0}
	b.rowCache = make(map[int]*Row) // キャッシュをクリア
}

// GetContent は指定された行の内容を返す
func (b *Buffer) GetContent(lineNum int) string {
	if lineNum < 0 || lineNum >= len(b.lines) {
		return ""
	}

	// 行のキャッシュを更新
	row := b.getRow(lineNum)
	if row == nil {
		return ""
	}

	return row.GetContent()
}

// GetAllContent はバッファの全内容を返す
func (b *Buffer) GetAllContent() []string {
	return b.lines
}

// InsertChar は現在のカーソル位置に文字を挿入する
func (b *Buffer) InsertChar(ch rune) {
	// 空のバッファの場合、最初の行を作成
	if len(b.lines) == 0 {
		b.lines = append(b.lines, "")
		b.rowCache = make(map[int]*Row)
	}

	// 現在の行のRowオブジェクトを取得
	row := b.getRow(b.cursor.Y)
	if row == nil {
		return
	}

	// 現在の行をルーンスライスに変換
	runes := []rune(row.GetContent())

	// カーソル位置が範囲外の場合は調整
	if b.cursor.X > len(runes) {
		b.cursor.X = len(runes)
	}

	// 文字を挿入（ルーンスライスを使用）
	newRunes := make([]rune, 0, len(runes)+1)
	newRunes = append(newRunes, runes[:b.cursor.X]...)
	newRunes = append(newRunes, ch)
	newRunes = append(newRunes, runes[b.cursor.X:]...)

	// 行を更新
	b.lines[b.cursor.Y] = string(newRunes)
	delete(b.rowCache, b.cursor.Y)

	// カーソル位置を更新
	b.cursor.X++
	b.isDirty = true
}

// DeleteChar はカーソル位置の文字を削除する
func (b *Buffer) DeleteChar() {
	if len(b.lines) == 0 || b.cursor.Y >= len(b.lines) {
		return
	}

	// 現在の行の取得
	if b.cursor.X == 0 {
		// カーソルが行頭の場合
		if b.cursor.Y > 0 {
			// 前の行に結合する
			prevLine := b.lines[b.cursor.Y-1]
			currLine := b.lines[b.cursor.Y]

			// 前の行と現在の行を結合
			b.lines[b.cursor.Y-1] = prevLine + currLine

			// 行を削除（スライスを縮める）
			if b.cursor.Y < len(b.lines)-1 {
				// 後ろの行を前に詰める
				b.lines = append(b.lines[:b.cursor.Y], b.lines[b.cursor.Y+1:]...)
			} else {
				// 最終行の場合は単純に切り詰める
				b.lines = b.lines[:b.cursor.Y]
			}

			// カーソル位置を更新
			b.cursor.Y--
			b.cursor.X = len([]rune(prevLine))

			// キャッシュをクリア
			b.rowCache = make(map[int]*Row)
			b.isDirty = true
		}
	} else {
		// 行の途中の場合
		currLine := b.lines[b.cursor.Y]
		runes := []rune(currLine)

		if b.cursor.X <= len(runes) {
			// カーソル位置の文字を削除
			b.cursor.X--
			b.lines[b.cursor.Y] = string(append(runes[:b.cursor.X], runes[b.cursor.X+1:]...))
			delete(b.rowCache, b.cursor.Y)
			b.isDirty = true
		}
	}
}

// getRow は指定された行のRowオブジェクトを取得する
func (b *Buffer) getRow(y int) *Row {
	if y < 0 || y >= len(b.lines) {
		return nil
	}

	if row, ok := b.rowCache[y]; ok {
		return row
	}

	row := NewRow(b.lines[y])
	b.rowCache[y] = row
	return row
}

// MoveCursor はカーソルを移動する
func (b *Buffer) MoveCursor(movement CursorMovement) {
	if len(b.lines) == 0 {
		return
	}
	currentRow := b.getRow(b.cursor.Y)
	if currentRow == nil {
		return
	}

	switch movement {
	case CursorLeft:
		if b.cursor.X > 0 {
			b.cursor.X--
		} else if b.cursor.Y > 0 {
			b.cursor.Y--
			targetRow := b.getRow(b.cursor.Y)
			if targetRow != nil {
				b.cursor.X = targetRow.GetRuneCount()
			}
		}
	case CursorRight:
		maxX := currentRow.GetRuneCount()
		if b.cursor.X < maxX {
			b.cursor.X++
		} else if b.cursor.Y < len(b.lines)-1 {
			b.cursor.Y++
			b.cursor.X = 0
		}
	case CursorUp:
		if b.cursor.Y > 0 {
			// 現在のカーソル位置の表示位置を取得
			currentScreenPos := currentRow.OffsetToScreenPosition(b.cursor.X)

			// 上の行に移動
			b.cursor.Y--
			targetRow := b.getRow(b.cursor.Y)
			if targetRow != nil {
				// 同じ表示位置に最も近い文字位置を探す
				b.cursor.X = targetRow.ScreenPositionToOffset(currentScreenPos)
			}
		}
	case CursorDown:
		if b.cursor.Y < len(b.lines)-1 {
			// 現在のカーソル位置の表示位置を取得
			currentScreenPos := currentRow.OffsetToScreenPosition(b.cursor.X)

			// 下の行に移動
			b.cursor.Y++
			targetRow := b.getRow(b.cursor.Y)
			if targetRow != nil {
				// 同じ表示位置に最も近い文字位置を探す
				b.cursor.X = targetRow.ScreenPositionToOffset(currentScreenPos)
			}
		}
	}
}

// SetCursor は指定された位置にカーソルを設定する
func (b *Buffer) SetCursor(x, y int) {
	if y >= 0 && y < len(b.lines) {
		row := b.getRow(y)
		if row != nil {
			b.cursor.Y = y
			if x >= 0 && x <= row.GetRuneCount() {
				b.cursor.X = x
			} else {
				b.cursor.X = row.GetRuneCount()
			}
		}
	}
}

// GetCursorXY はカーソル位置をx,y座標として返す
func (b *Buffer) GetCursorXY() (x, y int) {
	return b.cursor.X, b.cursor.Y
}

// InsertNewline は現在のカーソル位置で改行を挿入する
func (b *Buffer) InsertNewline() {
	// 空のバッファの場合、新しい行を追加
	if len(b.lines) == 0 {
		b.lines = append(b.lines, "", "")
		b.cursor.Y = 1
		b.cursor.X = 0
		b.isDirty = true
		return
	}

	currentLine := b.lines[b.cursor.Y]
	currentRunes := []rune(currentLine)

	// 現在の行を分割
	var firstPart, secondPart string
	if b.cursor.X <= len(currentRunes) {
		firstPart = string(currentRunes[:b.cursor.X])
		if b.cursor.X < len(currentRunes) {
			secondPart = string(currentRunes[b.cursor.X:])
		}
	} else {
		firstPart = currentLine
		secondPart = ""
	}

	// 元の行を更新
	b.lines[b.cursor.Y] = firstPart

	// 新しい行を挿入するためのスペースを確保
	b.lines = append(b.lines, "")                        // 一時的に末尾に空の行を追加
	copy(b.lines[b.cursor.Y+2:], b.lines[b.cursor.Y+1:]) // 後続の行を1つ後ろにシフト
	b.lines[b.cursor.Y+1] = secondPart                   // 分割した後半を新しい行として挿入

	// カーソルを次の行の先頭に移動
	b.cursor.Y++
	b.cursor.X = 0
	b.isDirty = true

	// 関連する行のキャッシュを更新
	for i := b.cursor.Y - 1; i < len(b.lines); i++ {
		delete(b.rowCache, i)
	}
}

// GetCursor は現在のカーソル位置を返す
func (b *Buffer) GetCursor() Cursor {
	return b.cursor
}

// GetCharAtCursor は現在のカーソル位置の文字を返す
func (b *Buffer) GetCharAtCursor() string {
	row := b.getRow(b.cursor.Y)
	if row == nil {
		return ""
	}

	// カーソル位置が有効か確認
	r, ok := row.GetRuneAt(b.cursor.X)
	if !ok {
		return ""
	}

	return string(r)
}

// GetLineCount は行数を返す
func (b *Buffer) GetLineCount() int {
	if b.lines == nil {
		return 0
	}
	return len(b.lines)
}

// IsDirty は未保存の変更があるかどうかを返す
func (b *Buffer) IsDirty() bool {
	return b.isDirty
}

// SetDirty はダーティフラグを設定する
func (b *Buffer) SetDirty(dirty bool) {
	b.isDirty = dirty
}

// Row は1行のテキストデータと関連情報を保持する
type Row struct {
	chars      string // 実際の文字列データ
	runes      []rune // 文字列をルーンに分解したもの
	widths     []int  // 各文字の表示幅
	positions  []int  // 各文字の表示位置（累積幅）
	totalWidth int    // 行全体の表示幅
}

// NewRow は新しいRow構造体を作成する
func NewRow(chars string) *Row {
	r := &Row{
		chars: chars,
		runes: []rune(chars),
	}
	r.updateWidths()
	return r
}

// updateWidths は行の文字幅情報を更新する
func (r *Row) updateWidths() {
	r.runes = []rune(r.chars)
	r.widths = make([]int, len(r.runes))
	r.positions = make([]int, len(r.runes)+1)
	r.totalWidth = 0

	for i, ch := range r.runes {
		w := getCharWidth(ch)
		r.widths[i] = w
		r.positions[i] = r.totalWidth
		r.totalWidth += w
	}
	r.positions[len(r.runes)] = r.totalWidth // 最後の位置も記録
}

// ScreenPositionToOffset は画面上の位置から文字列中のオフセットを取得する
func (r *Row) ScreenPositionToOffset(screenPos int) int {
	if len(r.positions) == 0 {
		return 0
	}

	// 画面位置が行の最後を超える場合は最後の文字の位置を返す
	if screenPos >= r.totalWidth {
		return len(r.runes)
	}
	if screenPos < 0 {
		return 0
	}

	// 画面位置に最も近い文字位置を探す
	for i := 0; i < len(r.positions); i++ {
		// 現在の文字の表示開始位置
		currentPos := r.positions[i]
		// 次の文字の表示開始位置（最後の文字の場合は行の総幅を使用）
		nextPos := r.totalWidth
		if i < len(r.positions)-1 {
			nextPos = r.positions[i+1]
		}

		// 画面位置が現在の文字の範囲内にある場合
		if screenPos >= currentPos && screenPos < nextPos {
			// より近い方の文字位置を返す
			if i < len(r.positions)-1 && (nextPos-screenPos < screenPos-currentPos) {
				return i + 1
			}
			return i
		}
	}

	return len(r.runes)
}

// OffsetToScreenPosition は文字列中のオフセットから画面上の位置を取得する
func (r *Row) OffsetToScreenPosition(offset int) int {
	if offset < 0 {
		return 0
	}
	if offset >= len(r.runes) {
		return r.totalWidth
	}
	return r.positions[offset]
}

// GetRuneCount は行の文字数を返す
func (r *Row) GetRuneCount() int {
	return len(r.runes)
}

// GetContent は行の内容を文字列として返す
func (r *Row) GetContent() string {
	return r.chars
}

// GetRuneAt は指定された位置のルーンを返す
func (r *Row) GetRuneAt(offset int) (rune, bool) {
	if offset < 0 || offset >= len(r.runes) {
		return 0, false
	}
	return r.runes[offset], true
}

// GetRuneWidth は指定された位置の文字の表示幅を返す
func (r *Row) GetRuneWidth(offset int) int {
	if offset < 0 || offset >= len(r.widths) {
		return 0
	}
	return r.widths[offset]
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

// Append は行の末尾に文字列を追加する
func (r *Row) Append(s string) {
	r.chars += s
	r.updateWidths()
}

// GetPositionFromOffset はオフセット位置から実際の表示位置を返す
func (r *Row) GetPositionFromOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	if offset >= len(r.positions) {
		return r.totalWidth
	}
	return r.positions[offset]
}
