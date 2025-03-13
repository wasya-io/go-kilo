package contents

import (
	"strings"
)

type Builder interface {
	Clear()
	Write(s string)
	Build() string
}

type StandardContentsBuilder struct {
	builder strings.Builder
}

func NewBuilder() *StandardContentsBuilder {
	return &StandardContentsBuilder{
		builder: strings.Builder{},
	}
}

func (b *StandardContentsBuilder) Clear() {
	b.builder.Reset()
}

func (b *StandardContentsBuilder) Write(s string) {
	b.builder.WriteString(s)
}

func (b *StandardContentsBuilder) Build() string {
	return b.builder.String()
}
