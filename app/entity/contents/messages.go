package contents

import (
	"fmt"
	"time"
)

type Message interface {
	String() string
	Clear()
	SetMessage(format string, args ...interface{})
	Get() string
	GetTime() int64
}

type StandardMessage struct {
	Message     string
	Args        []interface{}
	MessageTime int64
}

type DebugMessage string

func NewMessage(message string, args ...interface{}) *StandardMessage {
	return &StandardMessage{
		Message:     message,
		Args:        args,
		MessageTime: 0,
	}
}

func (m *StandardMessage) String() string {
	return fmt.Sprintf(m.Message, m.Args...)
}

func (m *StandardMessage) Clear() {
	m.Message = ""
	m.Args = make([]interface{}, 0)
}

func (m *StandardMessage) SetMessage(format string, args ...interface{}) {
	m.Message = format
	m.Args = make([]interface{}, len(args))
	copy(m.Args, args)
	m.MessageTime = time.Now().Unix()
}

func (m *StandardMessage) Get() string {
	return m.Message
}

func (m *StandardMessage) GetTime() int64 {
	return m.MessageTime
}

func (d DebugMessage) String() string {
	return string(d)
}
