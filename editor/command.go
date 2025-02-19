package editor

// EditorCommand はエディタの操作を表すインターフェース
type EditorCommand interface {
	Execute() error
}

// EditorOperations はInputHandlerが必要とするエディタ操作を定義するインターフェース
type EditorOperations interface {
	InsertChar(ch rune)
	InsertChars(chars []rune) // 追加
	DeleteChar()
	MoveCursor(movement CursorMovement)
	InsertNewline()
	IsDirty() bool
	SetDirty(bool)
	SaveFile() error
	Quit()
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

func NewInsertCharCommand(editor EditorOperations, char rune) *InsertCharCommand {
	return &InsertCharCommand{editor: editor, char: char}
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

// QuitCommand は終了コマンド
type QuitCommand struct {
	editor EditorOperations
}

func NewQuitCommand(editor EditorOperations) *QuitCommand {
	return &QuitCommand{editor: editor}
}

func (c *QuitCommand) Execute() error {
	if c.editor.IsDirty() {
		c.editor.SetStatusMessage("Warning! File has unsaved changes. Press Ctrl-Q or Ctrl-C again to quit.")
		c.editor.SetDirty(false)
		return nil
	}
	c.editor.Quit()
	return nil
}

// SaveCommand は保存コマンド
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
