package editor

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
	MoveCursor(movement CursorMovement)
	SaveFile() error
	Quit()
	IsDirty() bool
	SetDirty(bool)
	SetStatusMessage(format string, args ...interface{})
	UpdateScroll()
	GetCursor() Cursor
	GetContent(lineNum int) string
	GetConfig() *Config
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
	c.editor.InsertChar(c.char)
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
	c.editor.InsertChars(c.chars)
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
	c.editor.DeleteChar()
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
	c.editor.InsertNewline()
	return nil
}

// MoveCursorCommand はカーソル移動コマンド
type MoveCursorCommand struct {
	editor   EditorOperations
	movement CursorMovement
}

func NewMoveCursorCommand(editor EditorOperations, movement CursorMovement) *MoveCursorCommand {
	return &MoveCursorCommand{editor: editor, movement: movement}
}

func (c *MoveCursorCommand) Execute() error {
	c.editor.MoveCursor(c.movement)
	c.editor.UpdateScroll()
	return nil
}

// SaveCommand はファイル保存コマンド
type SaveCommand struct {
	editor EditorOperations
}

func NewSaveCommand(editor EditorOperations) *SaveCommand {
	return &SaveCommand{editor: editor}
}

func (c *SaveCommand) Execute() error {
	if err := c.editor.SaveFile(); err != nil {
		c.editor.SetStatusMessage("Can't save! I/O error: %s", err)
		return err
	}
	return nil
}

// QuitCommand は終了コマンド
type QuitCommand struct {
	editor EditorOperations
}

func NewQuitCommand(editor EditorOperations) *QuitCommand {
	return &QuitCommand{editor: editor}
}

func (c *QuitCommand) Execute() error {
	// isDirtyフラグを変更せずに終了
	c.editor.Quit()
	return nil
}

// InsertTabCommand はタブ挿入コマンド
type InsertTabCommand struct {
	editor EditorOperations
}

func NewInsertTabCommand(editor EditorOperations) *InsertTabCommand {
	return &InsertTabCommand{editor: editor}
}

func (c *InsertTabCommand) Execute() error {
	// タブは空白に展開
	tabWidth := GetTabWidth()
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

func (c *UndentCommand) Execute() error {
	// 現在の行の先頭から適切な数の空白を削除
	tabWidth := GetTabWidth()
	for i := 0; i < tabWidth; i++ {
		c.editor.DeleteChar()
	}
	return nil
}
