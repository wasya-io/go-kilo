package contents

import (
	"fmt"

	"github.com/wasya-io/go-kilo/app/entity/core"
)

type (
	// Contents はテキストバッファを管理する構造体
	Contents struct {
		logger   core.Logger
		lines    []string
		isDirty  bool
		rowCache map[int]*Row
	}

	ContentsState struct {
		Content string   // 影響を受けた行の内容
		IsDirty bool     // 未保存の変更があるか
		Lines   []string // 影響を受けた行の範囲
	}
)

// NewContents は新しいBufferインスタンスを作成する
func NewContents(logger core.Logger) *Contents {
	return &Contents{
		logger:   logger,
		lines:    make([]string, 0),
		isDirty:  false,
		rowCache: make(map[int]*Row),
		// eventManager: eventManager,
	}
}

// LoadContent はバッファに内容をロードする
func (b *Contents) LoadContent(lines []string) {
	// prevState := b.getCurrentState()

	b.lines = lines
	b.isDirty = false
	b.rowCache = make(map[int]*Row)
}

// GetContentLine は指定行の内容を取得する
func (b *Contents) GetContentLine(lineNum int) string {
	if lineNum >= 0 && lineNum < len(b.lines) {
		return b.lines[lineNum]
	}
	return ""
}

// GetAllLines はバッファの全内容を[]string形式で取得する
func (b *Contents) GetAllLines() []string {
	return append([]string{}, b.lines...)
}

// InsertChar は指定位置に文字を挿入する
func (b *Contents) InsertChar(pos Position, ch rune) {
	// prevState := b.getCurrentState()

	// 空のバッファの場合、最初の行を作成
	if len(b.lines) == 0 {
		b.lines = append(b.lines, "")
		b.rowCache = make(map[int]*Row)
	}

	// 指定位置の行のRowオブジェクトを取得
	row := b.GetRow(pos.Y)
	if row == nil {
		return
	}

	// 文字を挿入
	row.InsertChar(pos.X, ch)
	b.lines[pos.Y] = row.GetContent()
	delete(b.rowCache, pos.Y)
	b.isDirty = true

	// イベントを発行
	// b.publishBufferEvent(events.BufferContentChanged, pos, ch, prevState)
}

// InsertChars は複数の文字を一度に挿入する
func (b *Contents) InsertChars(pos Position, chars []rune) {
	if len(chars) == 0 {
		return
	}

	// prevState := b.getCurrentState()

	// 空のバッファの場合、最初の行を作成
	if len(b.lines) == 0 {
		b.lines = append(b.lines, "")
		b.rowCache = make(map[int]*Row)
	}

	// 指定位置の行のRowオブジェクトを取得
	row := b.GetRow(pos.Y)
	if row == nil {
		return
	}

	// すべての文字を指定位置の行に挿入
	for _, ch := range chars {
		row.InsertChar(pos.X, ch)
		pos.X++
	}

	// 行の内容を更新
	b.lines[pos.Y] = row.GetContent()
	delete(b.rowCache, pos.Y)
	b.isDirty = true

	// 一度だけイベントを発行
	// b.publishBufferEvent(events.BufferContentChanged, pos, chars, prevState)
}

// DeleteChar は指定位置の文字を削除する
func (b *Contents) DeleteChar(pos Position) {
	if len(b.lines) == 0 || pos.Y >= len(b.lines) {
		return
	}

	// prevState := b.getCurrentState()

	// カーソルが行頭にある場合
	if pos.X == 0 {
		if pos.Y > 0 {
			// 前の行に結合する処理
			prevLine := b.lines[pos.Y-1]
			currLine := b.lines[pos.Y]

			// 行を結合（現在の行が空でない場合のみ）
			if currLine != "" {
				b.lines[pos.Y-1] = prevLine + currLine
			}

			// 現在の行より後ろの行をすべて1つ前にシフト
			copy(b.lines[pos.Y:], b.lines[pos.Y+1:])
			// スライスの長さを1つ減らす
			b.lines = b.lines[:len(b.lines)-1]

			// キャッシュをクリア
			for i := pos.Y - 1; i < len(b.lines); i++ {
				delete(b.rowCache, i)
			}
			b.isDirty = true
		}
	} else {
		// カーソル位置の前の文字を削除
		row := b.GetRow(pos.Y)
		if row != nil && pos.X > 0 {
			row.DeleteChar(pos.X - 1)
			b.lines[pos.Y] = row.GetContent()
			delete(b.rowCache, pos.Y)
			b.isDirty = true
		}
	}

	// イベントを発行
	// b.publishBufferEvent(events.BufferContentChanged, pos, nil, prevState)
}

// InsertNewline は指定位置で改行を挿入する
func (b *Contents) InsertNewline(pos Position) {
	// prevState := b.getCurrentState()

	// 空のバッファの場合、新しい行を追加
	if len(b.lines) == 0 {
		b.lines = append(b.lines, "", "")
		b.isDirty = true
		// イベントを発行
		// b.publishBufferEvent(events.BufferStructuralChange, pos, nil, prevState)
		return
	}

	currentLine := b.lines[pos.Y]
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
	b.lines[pos.Y] = firstPart

	// 新しい行を挿入するためのスペースを確保
	b.lines = append(b.lines, "")
	copy(b.lines[pos.Y+2:], b.lines[pos.Y+1:])
	b.lines[pos.Y+1] = secondPart

	b.isDirty = true

	// 関連する行のキャッシュを更新
	for i := pos.Y; i < len(b.lines); i++ {
		delete(b.rowCache, i)
	}

	// 構造的な変更を通知
	// b.publishBufferEvent(events.BufferStructuralChange, pos, nil, prevState)
}

// GetLineCount は行数を返す
func (b *Contents) GetLineCount() int {
	if b.lines == nil {
		return 0
	}
	return len(b.lines)
}

// IsDirty は未保存の変更があるかどうかを返す
func (b *Contents) IsDirty() bool {
	return b.isDirty
}

// SetDirty はダーティフラグを設定する
func (b *Contents) SetDirty(dirty bool) {
	if b.isDirty != dirty {
		// prevState := b.getCurrentState()
		b.isDirty = dirty
		// b.publishBufferEvent(events.BufferContentChanged, events.Position{}, dirty, prevState)
	}
}

// RestoreState は以前の状態にバッファを復元する
func (b *Contents) RestoreState(state interface{}) error {
	fmt.Printf("Debug: Buffer.RestoreState called with state type: %T\n", state)

	if bufferState, ok := state.(ContentsState); ok {
		fmt.Printf("Debug: Restoring buffer state: Content=%q, IsDirty=%v, Lines=%v\n",
			bufferState.Content, bufferState.IsDirty, bufferState.Lines)

		// // イベントマネージャを一時的に保存して無効化
		// tempManager := b.eventManager
		// b.eventManager = nil

		// バッファを完全にクリア
		b.lines = []string{""}
		b.rowCache = make(map[int]*Row)
		b.isDirty = false

		// 新しい状態を直接設定
		if len(bufferState.Lines) > 0 {
			b.lines = make([]string, len(bufferState.Lines))
			copy(b.lines, bufferState.Lines)
		} else if bufferState.Content != "" {
			b.lines = []string{bufferState.Content}
		}

		// 行キャッシュを再構築
		b.rowCache = make(map[int]*Row)
		for i := range b.lines {
			b.GetRow(i)
		}

		// ダーティフラグを設定
		b.isDirty = bufferState.IsDirty

		// // イベントマネージャを復元し、状態変更イベントを発行
		// b.eventManager = tempManager
		// if b.eventManager != nil {
		// 	prevState := events.BufferState{Content: "", IsDirty: false, Lines: []string{""}}
		// 	b.publishBufferEvent(events.BufferEventSetState, events.Position{}, bufferState, prevState)
		// }

		fmt.Printf("Debug: Buffer state restored. Final content: %q\n", b.lines)
		return nil
	}
	return fmt.Errorf("invalid state type for buffer restoration: %T", state)
}

// GetRow は指定された行のRowオブジェクトを取得する
func (b *Contents) GetRow(y int) *Row {
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

// getCurrentState は現在のバッファ状態を取得する
func (b *Contents) GetCurrentState() ContentsState {
	// 完全な状態をコピーして返す
	state := ContentsState{
		Content: "",
		IsDirty: b.isDirty,
		Lines:   make([]string, len(b.lines)),
	}

	// 内容をコピー
	if len(b.lines) > 0 {
		state.Content = b.lines[0] // 最初の行を Content フィールドに設定
		copy(state.Lines, b.lines) // すべての行を Lines フィールドにコピー
	}

	return state
}

// resetToCleanState はバッファを初期状態にリセットする
func (b *Contents) Initialize() error {
	// 現在の状態を保存
	// prevState := b.getCurrentState()

	// バッファを完全にリセット
	b.lines = []string{""}
	b.rowCache = make(map[int]*Row)
	b.isDirty = false

	// リセットイベントを発行
	// if b.eventManager != nil {
	// 	b.publishBufferEvent(events.BufferEventSetState,
	// 		events.Position{}, nil, prevState)
	// }

	return nil
}
