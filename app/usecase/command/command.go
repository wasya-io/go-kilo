package command

type (
	Command interface {
		Execute() error
	}

	StandardCommand struct {
		fn func() error
	}

	InsertCharCommand struct {
		char rune
		fn   func(rune) error
	}
)

func NewCommand(execute func() error) StandardCommand {
	return StandardCommand{fn: execute}
}

func (c StandardCommand) Execute() error {
	return c.fn()
}

func NewInsertCharCommand(char rune, execute func(rune) error) InsertCharCommand {
	return InsertCharCommand{char: char, fn: execute}
}

func (c InsertCharCommand) Execute() error {
	return c.fn(c.char)
}
