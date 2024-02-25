package guacamole

import (
	"errors"
	"github.com/gorilla/websocket"
)

type TunnelState int

const (
	TunnelClosed TunnelState = iota
	TunnelOpen
)

const Delimiter = ';'
const Version = "VERSION_1_5_3"

var ErrNotConnected = errors.New("not connected")

type Tunnel interface {
	Address() string
	Connect() error
	Disconnect()
	Send(raw []byte) error
	Receive() ([]byte, error)
	SendInstruction(ins ...*Instruction) error
	ReceiveInstruction() (*Instruction, error)
	UUID() string

	To(ws *websocket.Conn) error
	Join(guest string) Tunnel
	Quit(guest string)
	Close()
}
