package input

import (
	"fmt"

	"github.com/wasya-io/go-kilo/app/boundary/reader"
	"github.com/wasya-io/go-kilo/app/entity/core"
	"github.com/wasya-io/go-kilo/app/entity/key"
	"github.com/wasya-io/go-kilo/app/usecase/parser"
)

type Provider interface {
	GetInputEvents() (key.KeyEvent, []key.KeyEvent, error)
}

type StandardInputProvider struct {
	logger core.Logger
	reader reader.KeyReader
	parser parser.InputParser
}

func NewStandardInputProvider(logger core.Logger, reader reader.KeyReader, parser parser.InputParser) *StandardInputProvider {
	return &StandardInputProvider{
		logger: logger,
		reader: reader,
		parser: parser,
	}
}

func (p *StandardInputProvider) GetInputEvents() (key.KeyEvent, []key.KeyEvent, error) {
	buf, n, err := p.reader.Read()
	if err != nil {
		return key.KeyEvent{}, nil, fmt.Errorf("input error: %v", err)
	}
	if n == 0 {
		return key.KeyEvent{}, nil, fmt.Errorf("no input")
	}
	events, err := p.parser.Parse(buf, n)
	if err != nil {
		return key.KeyEvent{}, nil, fmt.Errorf("input error: %v", err)
	}
	p.logger.ReadyWithType("GetInputEvents").WithType().WithString().WithString().Do(events[0], events[0].Type, events[0].Rune)
	return events[0], events[1:], nil
}
