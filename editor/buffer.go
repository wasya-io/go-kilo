package editor

import (
	"fmt"

	"github.com/wasya-io/go-kilo/editor/events"
)

// Buffer はテキストバッファを管理する構造体
type Buffer struct {
	content      []string
	isDirty      bool
	rowCache     map[int]*Row
	eventManager *events.EventManager
	Filename     string // ファイル名を追加
}

// NewBuffer は新しいBufferインスタンスを作成する
func NewBuffer(eventManager *events.EventManager) *Buffer {
	return &Buffer{
		content:      make([]string, 0),
		isDirty:      false,
		rowCache:     make(map[int]*Row),
		eventManager: eventManager,
	}
}

// LoadContent はバッファに内容をロードする
func (b *Buffer) LoadContent(lines []string) {
	prevState := b.getCurrentState()

	b.content = lines
	b.isDirty = false
	b.rowCache = make(map[int]*Row)

	// バッファ内容変更イベントを発行
	b.publishBufferEvent(events.BufferContentChanged, events.Position{}, lines, prevState)
}

// GetContentLine は指定行の内容を取得する
func (b *Buffer) GetContentLine(lineNum int) string {
	if lineNum >= 0 && lineNum < len(b.content) {
		return b.content[lineNum]
	}
	return ""
}

// GetAllLines はバッファの全内容を[]string形式で取得する
func (b *Buffer) GetAllLines() []string {
	return append([]string{}, b.content...)
}

// GetAllContent はバッファの全内容を文字列として取得する
func (b *Buffer) GetAllContent() string {
	content := ""
	for i, line := range b.content {
		if i > 0 {
			content += "\n"
		}
		content += line
	}
	return content
}

// InsertChar は指定位置に文字を挿入する
func (b *Buffer) InsertChar(pos events.Position, ch rune) {
	prevState := b.getCurrentState()

	// 空のバッファの場合、最初の行を作成
	if len(b.content) == 0 {
		b.content = append(b.content, "")
		b.rowCache = make(map[int]*Row)
	}

	// 指定位置の行のRowオブジェクトを取得
	row := b.getRow(pos.Y)
	if row == nil {
		return
	}

	// 文字を挿入
	row.InsertChar(pos.X, ch)
	b.content[pos.Y] = row.GetContent()
	delete(b.rowCache, pos.Y)
	b.isDirty = true

	// イベントを発行
	b.publishBufferEvent(events.BufferContentChanged, pos, ch, prevState)
}

// InsertChars は複数の文字を一度に挿入する
func (b *Buffer) InsertChars(pos events.Position, chars []rune) {
	if len(chars) == 0 {
		return
	}

	prevState := b.getCurrentState()

	// 空のバッファの場合、最初の行を作成
	if len(b.content) == 0 {
		b.content = append(b.content, "")
		b.rowCache = make(map[int]*Row)
	}

	// 指定位置の行のRowオブジェクトを取得
	row := b.getRow(pos.Y)
	if row == nil {
		return
	}

	// すべての文字を指定位置の行に挿入
	for _, ch := range chars {
		row.InsertChar(pos.X, ch)
		pos.X++
	}

	// 行の内容を更新
	b.content[pos.Y] = row.GetContent()
	delete(b.rowCache, pos.Y)
	b.isDirty = true

	// 一度だけイベントを発行
	b.publishBufferEvent(events.BufferContentChanged, pos, chars, prevState)
}

// DeleteChar は指定位置の文字を削除する
func (b *Buffer) DeleteChar(pos events.Position) {
	if len(b.content) == 0 || pos.Y >= len(b.content) {
		return
	}

	prevState := b.getCurrentState()

	// カーソルが行頭にある場合
	if pos.X == 0 {
		if pos.Y > 0 {
			// 前の行に結合する処理
			prevLine := b.content[pos.Y-1]
			currLine := b.content[pos.Y]

			// 行を結合（現在の行が空でない場合のみ）
			if currLine != "" {
				b.content[pos.Y-1] = prevLine + currLine
			}

			// 現在の行より後ろの行をすべて1つ前にシフト
			copy(b.content[pos.Y:], b.content[pos.Y+1:])
			// スライスの長さを1つ減らす
			b.content = b.content[:len(b.content)-1]

			// キャッシュをクリア
			for i := pos.Y - 1; i < len(b.content); i++ {
				delete(b.rowCache, i)
			}
			b.isDirty = true
		}
	} else {
		// カーソル位置の前の文字を削除
		row := b.getRow(pos.Y)
		if row != nil && pos.X > 0 {
			row.DeleteChar(pos.X - 1)
			b.content[pos.Y] = row.GetContent()
			delete(b.rowCache, pos.Y)
			b.isDirty = true
		}
	}

	// イベントを発行
	b.publishBufferEvent(events.BufferContentChanged, pos, nil, prevState)
}

// getRow は指定された行のRowオブジェクトを取得する
func (b *Buffer) getRow(y int) *Row {
	if y < 0 || y >= len(b.content) {
		return nil
	}

	if row, ok := b.rowCache[y]; ok && row != nil {
		return row
	}

	row := NewRow(b.content[y])
	b.rowCache[y] = row
	return row
}

// InsertNewline は指定位置で改行を挿入する
func (b *Buffer) InsertNewline(pos events.Position) {
	prevState := b.getCurrentState()

	// 空のバッファの場合、新しい行を追加
	if len(b.content) == 0 {
		b.content = append(b.content, "", "")
		b.isDirty = true
		// イベントを発行
		b.publishBufferEvent(events.BufferStructuralChange, pos, nil, prevState)
		return
	}

	currentLine := b.content[pos.Y]
	currentRunes := []rune(currentLine)

	// 現在の行を分割
	var firstPart, secondPart string
	if pos.X <= len(currentRunes) {
		firstPart = string(currentRunes[:pos.X])
		if pos.X < len(currentRunes) {
			secondPart = string(currentRunes[pos.X:])
		}
	} else {
		firstPart = currentLine
		secondPart = ""
	}

	// 元の行を更新
	b.content[pos.Y] = firstPart

	// 新しい行を挿入するためのスペースを確保
	b.content = append(b.content, "")
	copy(b.content[pos.Y+2:], b.content[pos.Y+1:])
	b.content[pos.Y+1] = secondPart

	b.isDirty = true

	// 関連する行のキャッシュを更新
	for i := pos.Y; i < len(b.content); i++ {
		delete(b.rowCache, i)
	}

	// 構造的な変更を通知
	b.publishBufferEvent(events.BufferStructuralChange, pos, nil, prevState)
}

// GetLineCount は行数を返す
func (b *Buffer) GetLineCount() int {
	if b.content == nil {
		return 0
	}
	return len(b.content)
}

// IsDirty は未保存の変更があるかどうかを返す
func (b *Buffer) IsDirty() bool {
	return b.isDirty
}

// SetDirty はダーティフラグを設定する
func (b *Buffer) SetDirty(dirty bool) {
	if b.isDirty != dirty {
		prevState := b.getCurrentState()
		b.isDirty = dirty
		b.publishBufferEvent(events.BufferContentChanged, events.Position{}, dirty, prevState)
	}
}

// getCurrentState は現在のバッファ状態を取得する
func (b *Buffer) getCurrentState() events.BufferState {
	var content string
	var lines []string
	if len(b.content) > 0 {
		content = b.content[0]
		lines = b.content[:1] // 最初の行のみを含める
	}
	return events.BufferState{
		Content: content,
		IsDirty: b.isDirty,
		Lines:   lines,
	}
}

// RestoreState は以前の状態にバッファを復元する
func (b *Buffer) RestoreState(state interface{}) error {
	if bufferState, ok := state.(events.BufferState); ok {
		b.content = bufferState.Lines
		b.isDirty = bufferState.IsDirty
		return nil
	}
	return fmt.Errorf("invalid state type for buffer restoration")
}

// publishBufferEvent はバッファイベントを発行する
func (b *Buffer) publishBufferEvent(eventType events.BufferEventSubType, editPos events.Position, data interface{}, prevState events.BufferState) {
	if b.eventManager == nil {
		return
	}

	currentState := b.getCurrentState()
	event := events.NewBufferEvent(eventType, data)
	event.SetStates(prevState, currentState)

	// 編集位置の情報をBufferChangeDataに設定
	if editPos.Y >= 0 && editPos.Y < len(b.content) {
		change := events.BufferChangeData{
			StartLine: editPos.Y,
			EndLine:   editPos.Y,
		}
		event.AddChange(change)
	}

	b.eventManager.Publish(event)
}

// publishPartialRefreshEvent は部分更新イベントを発行する
func (b *Buffer) publishPartialRefreshEvent(lines []int) {
	if b.eventManager == nil {
		return
	}

	data := events.EditorUpdateData{
		Lines:    lines,
		ForceAll: false,
	}
	event := events.NewUIEvent(events.UIEditorPartialRefresh, data)
	b.eventManager.Publish(event)
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
