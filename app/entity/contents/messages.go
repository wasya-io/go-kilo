package contents

import (
	"fmt"
	"time"
)

type Message struct {
	Message     string
	Args        []interface{}
	MessageTime int64
}

type DebugMessage string

func NewMessage(message string, args ...interface{}) *Message {
	return &Message{
		Message:     message,
		Args:        args,
		MessageTime: 0,
	}
}

func (m *Message) String() string {
	return fmt.Sprintf(m.Message, m.Args...)
}

func (m *Message) Clear() {
	m.Message = ""
	m.Args = make([]interface{}, 0)
}

func (m *Message) SetMessage(format string, args ...interface{}) {
	m.Message = format
	m.Args = make([]interface{}, len(args))
	copy(m.Args, args)
	m.MessageTime = time.Now().Unix()
}
