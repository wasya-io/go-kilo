package command

type (
	Command interface {
		Execute() error
	}

	StandardCommand struct {
		fn func() error
	}
)

func NewCommand(execute func() error) StandardCommand {
	return StandardCommand{fn: execute}
}

func (c StandardCommand) Execute() error {
	return c.fn()
}
