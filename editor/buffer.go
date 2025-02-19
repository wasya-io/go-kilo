package editor

import "github.com/wasya-io/go-kilo/editor/events"

// Buffer はテキストバッファを管理する構造体
type Buffer struct {
	lines        []string
	cursor       Cursor
	isDirty      bool
	rowCache     map[int]*Row
	eventManager *events.EventManager
}

// Cursor はカーソル位置を管理する構造体
type Cursor struct {
	X, Y int
}

// NewBuffer は新しいBufferインスタンスを作成する
func NewBuffer(eventManager *events.EventManager) *Buffer {
	return &Buffer{
		lines:        make([]string, 0),
		cursor:       Cursor{X: 0, Y: 0},
		isDirty:      false,
		rowCache:     make(map[int]*Row),
		eventManager: eventManager,
	}
}

// LoadContent はバッファに内容をロードする
func (b *Buffer) LoadContent(lines []string) {
	prevState := b.getCurrentState()

	b.lines = lines
	b.isDirty = false
	b.cursor = Cursor{X: 0, Y: 0}
	b.rowCache = make(map[int]*Row)

	// バッファ内容変更イベントを発行
	b.publishBufferEvent(events.BufferLoadContent, b.cursor, nil, prevState)
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
	prevState := b.getCurrentState()

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

	// 文字を挿入
	row.InsertChar(b.cursor.X, ch)
	b.lines[b.cursor.Y] = row.GetContent()
	delete(b.rowCache, b.cursor.Y)
	b.cursor.X++
	b.isDirty = true

	// イベントを発行
	b.publishBufferEvent(events.BufferInsertChar, b.cursor, ch, prevState)
}

// InsertChars は複数の文字を一度に挿入する
func (b *Buffer) InsertChars(chars []rune) {
	if len(chars) == 0 {
		return
	}

	prevState := b.getCurrentState()

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

	// すべての文字を現在の行に挿入
	for _, ch := range chars {
		row.InsertChar(b.cursor.X, ch)
		b.cursor.X++
	}

	// 行の内容を更新
	b.lines[b.cursor.Y] = row.GetContent()
	delete(b.rowCache, b.cursor.Y)
	b.isDirty = true

	// 一度だけイベントを発行
	b.publishBufferEvent(events.BufferInsertChar, b.cursor, chars, prevState)
}

// DeleteChar はカーソル位置の文字を削除する
func (b *Buffer) DeleteChar() {
	if len(b.lines) == 0 || b.cursor.Y >= len(b.lines) {
		return
	}

	// カーソルが行頭にある場合
	if b.cursor.X == 0 {
		if b.cursor.Y > 0 {
			// 前の行に結合する処理
			prevLine := b.lines[b.cursor.Y-1]
			currLine := b.lines[b.cursor.Y]
			// カーソルは前の行の末尾に移動
			b.cursor.Y--
			b.cursor.X = len([]rune(prevLine))
			// 行を結合
			b.lines[b.cursor.Y] = prevLine + currLine
			// 現在の行を削除
			b.lines = append(b.lines[:b.cursor.Y+1], b.lines[b.cursor.Y+2:]...)
			// キャッシュをクリア
			delete(b.rowCache, b.cursor.Y)
			delete(b.rowCache, b.cursor.Y+1)
			b.isDirty = true
		}
	} else {
		// カーソル位置の前の文字を削除
		row := b.getRow(b.cursor.Y)
		if row != nil && b.cursor.X > 0 {
			row.DeleteChar(b.cursor.X - 1)
			b.lines[b.cursor.Y] = row.GetContent()
			delete(b.rowCache, b.cursor.Y)
			b.cursor.X--
			b.isDirty = true
		}
	}
}

// getRow は指定された行のRowオブジェクトを取得する
func (b *Buffer) getRow(y int) *Row {
	if y < 0 || y >= len(b.lines) {
		return nil
	}

	if row, ok := b.rowCache[y]; ok && row != nil {
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
			// カーソルを1文字分左に移動
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
			// カーソルを1文字分右に移動
			b.cursor.X++
		} else if b.cursor.Y < len(b.lines)-1 {
			b.cursor.Y++
			b.cursor.X = 0
		}
	case CursorUp:
		if b.cursor.Y > 0 {
			b.cursor.Y--
			targetRow := b.getRow(b.cursor.Y)
			if targetRow != nil {
				b.cursor.X = min(b.cursor.X, targetRow.GetRuneCount())
			}
		}
	case CursorDown:
		if b.cursor.Y < len(b.lines)-1 {
			b.cursor.Y++
			targetRow := b.getRow(b.cursor.Y)
			if targetRow != nil {
				b.cursor.X = min(b.cursor.X, targetRow.GetRuneCount())
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

// getCurrentState は現在のバッファ状態を取得する
func (b *Buffer) getCurrentState() events.BufferState {
	var content string
	if b.cursor.Y < len(b.lines) {
		content = b.lines[b.cursor.Y]
	}
	return events.BufferState{
		Content: content,
		IsDirty: b.isDirty,
		CursorPos: events.Position{
			X: b.cursor.X,
			Y: b.cursor.Y,
		},
	}
}

// publishBufferEvent はバッファイベントを発行する
func (b *Buffer) publishBufferEvent(op events.BufferOperationType, pos Cursor, data interface{}, prevState events.BufferState) {
	if b.eventManager == nil {
		return
	}

	currentState := b.getCurrentState()
	// 状態が実際に変更された場合のみイベントを発行
	if prevState != currentState {
		event := events.NewBufferEvent(
			op,
			events.Position{X: pos.X, Y: pos.Y},
			data,
			prevState,
			currentState,
		)
		b.eventManager.Publish(event)
	}
}

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

// GetContent は行の内容を文字列として返す
func (r *Row) GetContent() string {
	return r.chars
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
