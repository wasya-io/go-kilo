package editor

import (
	"github.com/wasya-io/go-kilo/app/config"
	"github.com/wasya-io/go-kilo/app/entity/cursor"
	"github.com/wasya-io/go-kilo/editor/events"
)

// Command はエディタのコマンドを表すインターフェース
type Command interface {
	Execute() error
}

// EditorOperations はコマンドが必要とするエディタの操作を定義する
type EditorOperations interface {
	InsertChar(ch rune)
	InsertChars(chars []rune) // 追加
	DeleteChar()
	InsertNewline()
	MoveCursor(movement cursor.Movement)
	SaveFile() error
	Quit()
	IsDirty() bool
	SetDirty(bool)
	SetStatusMessage(format string, args ...interface{})
	UpdateScroll()
	GetCursor() *cursor.Cursor
	GetContent(lineNum int) string
	GetConfig() *config.Config
}

// InsertCharCommand は文字挿入コマンド
type InsertCharCommand struct {
	editor EditorOperations
	char   rune
}

func NewInsertCharCommand(editor EditorOperations, ch rune) *InsertCharCommand {
	return &InsertCharCommand{editor: editor, char: ch}
}

func (c *InsertCharCommand) Execute() error {
	c.editor.InsertChar(c.char) // InsertChar get:ui, mod:buffer, mod:ui, call:RefreshScreen
	return nil
}

// InsertCharsCommand は複数の文字を一度に挿入するコマンド
type InsertCharsCommand struct {
	editor EditorOperations
	chars  []rune
}

func NewInsertCharsCommand(editor EditorOperations, chars ...rune) *InsertCharsCommand {
	return &InsertCharsCommand{editor: editor, chars: chars}
}

func (c *InsertCharsCommand) Execute() error {
	c.editor.InsertChars(c.chars) // InsertChars get:ui, mod:buffer, mod:ui
	return nil
}

// DeleteCharCommand は文字削除コマンド
type DeleteCharCommand struct {
	editor EditorOperations
}

func NewDeleteCharCommand(editor EditorOperations) *DeleteCharCommand {
	return &DeleteCharCommand{editor: editor}
}

func (c *DeleteCharCommand) Execute() error {
	c.editor.DeleteChar() // get:ui, mod:buffer, mod:ui, call:RefreshScreen
	return nil
}

// InsertNewlineCommand は改行挿入コマンド
type InsertNewlineCommand struct {
	editor EditorOperations
}

func NewInsertNewlineCommand(editor EditorOperations) *InsertNewlineCommand {
	return &InsertNewlineCommand{editor: editor}
}

func (c *InsertNewlineCommand) Execute() error {
	c.editor.InsertNewline() // get:ui, mod:buffer, set:ui
	return nil
}

// MoveCursorCommand はカーソル移動コマンド
type MoveCursorCommand struct {
	editor   EditorOperations
	movement cursor.Movement
}

func NewMoveCursorCommand(editor EditorOperations, movement cursor.Movement) *MoveCursorCommand {
	return &MoveCursorCommand{editor: editor, movement: movement}
}

func (c *MoveCursorCommand) Execute() error {
	c.editor.MoveCursor(c.movement) // mod:ui
	c.editor.UpdateScroll()         // get:ui, get:buffer, mod:ui
	return nil
}

// SaveCommand はファイル保存コマンド
type SaveCommand struct {
	editor interface {
		GetEventManager() *events.EventManager
		GetFilename() string
		IsDirty() bool
	}
}

func NewSaveCommand(editor interface {
	GetEventManager() *events.EventManager
	GetFilename() string
	IsDirty() bool
}) *SaveCommand {
	return &SaveCommand{editor: editor}
}

func (c *SaveCommand) Execute() error {
	// 変更がない場合は保存不要
	if !c.editor.IsDirty() {
		return nil
	}

	event := events.NewSaveEvent(c.editor.GetFilename(), false)
	return c.editor.GetEventManager().Publish(event)
}

// QuitCommand は終了コマンド
type QuitCommand struct {
	editor interface {
		GetEventManager() *events.EventManager
		IsDirty() bool
		IsQuitWarningShown() bool
	}
}

func NewQuitCommand(editor interface {
	GetEventManager() *events.EventManager
	IsDirty() bool
	IsQuitWarningShown() bool
}) *QuitCommand {
	return &QuitCommand{editor: editor}
}

func (c *QuitCommand) Execute() error {
	event := events.NewQuitEvent(
		c.editor.IsDirty(),
		c.editor.IsQuitWarningShown(),
	)
	return c.editor.GetEventManager().Publish(event)
}

// InsertTabCommand はタブ挿入コマンド
type InsertTabCommand struct {
	editor EditorOperations
}

func NewInsertTabCommand(editor EditorOperations) *InsertTabCommand {
	return &InsertTabCommand{editor: editor}
}

func (c *InsertTabCommand) Execute() error { // get:config, mod:buffer, mod:ui
	// タブは空白に展開
	tabWidth := config.GetTabWidth()
	for i := 0; i < tabWidth; i++ {
		c.editor.InsertChar(' ')
	}
	return nil
}

// UndentCommand はアンインデントコマンド
type UndentCommand struct {
	editor EditorOperations
}

func NewUndentCommand(editor EditorOperations) *UndentCommand {
	return &UndentCommand{editor: editor}
}

func (c *UndentCommand) Execute() error { // get:config, mod:buffer, mod:ui
	cur := c.editor.GetCursor()
	pos := cur.ToPosition()
	content := c.editor.GetContent(pos.Y)

	// カーソルが行頭にある場合は処理を行わない
	if pos.X <= 0 {
		return nil
	}

	// カーソル位置の左側のスペース数を数える
	leftSpaces := 0
	for i := pos.X - 1; i >= 0; i-- {
		if content[i] != ' ' {
			break
		}
		leftSpaces++
	}

	if leftSpaces == 0 {
		return nil // 左側にスペースがない場合は何もしない
	}

	// 削除するスペース数を計算
	tabWidth := config.GetTabWidth()
	spacesToDelete := leftSpaces % tabWidth
	if spacesToDelete == 0 {
		spacesToDelete = tabWidth
	}

	// カーソルを1つ左に移動し、削除を開始
	// c.editor.MoveCursor(CursorLeft)

	// スペースを削除
	for i := 0; i < spacesToDelete; i++ {
		c.editor.DeleteChar()
	}

	// 残りのカーソル移動
	for i := 1; i < (spacesToDelete - 1); i++ {
		c.editor.MoveCursor(cursor.CursorLeft)
	}

	return nil
}
