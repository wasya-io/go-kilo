package contents

import (
	"fmt"
	"strings"
)

type Builder interface {
	Clear()
	Write(s string)
	MoveCursor(row, col int) string
	Build() string
}

type StandardContentsBuilder struct {
	buffer strings.Builder
}

func NewBuilder() *StandardContentsBuilder {
	return &StandardContentsBuilder{
		buffer: strings.Builder{},
	}
}

func (b *StandardContentsBuilder) Clear() {
	b.buffer.Reset()
}

func (b *StandardContentsBuilder) Write(s string) {
	b.buffer.WriteString(s)
}

// moveCursor はカーソルを指定位置に移動する
func (b *StandardContentsBuilder) MoveCursor(row, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row+1, col+1)
}

func (b *StandardContentsBuilder) Build() string {
	return b.buffer.String()
}
