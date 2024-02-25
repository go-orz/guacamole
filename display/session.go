package guacenc

import (
	"github.com/go-orz/guacamole"
)

type SessionState int

const (
	SessionClosed SessionState = iota
	SessionHandshake
	SessionActive
)

type Session interface {
	Send(ins ...*guacamole.Instruction) error
	Read() <-chan *guacamole.Instruction
	Terminate()
	State() SessionState
}
