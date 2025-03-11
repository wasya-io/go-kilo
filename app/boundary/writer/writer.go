package writer

import (
	"os"
)

type ScreenWriter interface {
	Write(s string) error
}

type StandardScreenWriter struct {
}

func NewStandardScreenWriter() *StandardScreenWriter {
	return &StandardScreenWriter{}
}

func (w *StandardScreenWriter) Write(s string) error {
	_, err := os.Stdout.Write([]byte(s))
	return err
}
