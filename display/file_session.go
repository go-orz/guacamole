package guacenc

import (
	"context"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/go-orz/guacamole"
)

// Session is used to create and keep a connection with a guacd server,
// and it is responsible for the initial handshake and to send and receive instructions.
// Instructions received are put in the in channel. Instructions are sent using the Send() function
type fileSession struct {
	in     chan *guacamole.Instruction
	state  SessionState
	logger Logger
	file   *os.File
	io     *guacamole.InstructionIO
	ctx    context.Context
	cancel context.CancelFunc
}

// newNetSession creates a new connection with the guacd server, using the configuration provided
func newFileSession(path string, logger Logger) (*fileSession, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	instructionIO := guacamole.NewInstructionIO(file, false)
	ctx, cancel := context.WithCancel(context.Background())

	s := &fileSession{
		in:     make(chan *guacamole.Instruction),
		state:  SessionClosed,
		logger: logger,
		file:   file,
		io:     instructionIO,
		ctx:    ctx,
		cancel: cancel,
	}

	s.state = SessionActive
	go func() {
		defer instructionIO.Close()
		defer file.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				ins, err := instructionIO.Read()
				if err != nil {
					s.logger.Warnf("Disconnecting from server. Reason: " + err.Error())
					s.Terminate()
					break
				}
				//if ins.Opcode == "blob" {
				//	s.logger.Debugf("S> 4.blob, stream: %s, data len: %d", ins.Args[0], len(ins.Args[1]))
				//} else {
				//	s.logger.Debugf("S> %s", ins)
				//}
				if ins.Opcode == "nop" {
					continue
				}
				s.in <- ins
			}
		}
	}()
	return s, nil
}

// Terminate the current session, disconnecting from the server
func (s *fileSession) Terminate() {
	s.cancel()
	close(s.in)
}

// Send instructions to the server. Multiple instructions are sent in one single transaction
func (s *fileSession) Send(ins ...*guacamole.Instruction) error {

	return nil
}

func (s *fileSession) Read() <-chan *guacamole.Instruction {
	return s.in
}

func (s *fileSession) State() SessionState {
	return s.state
}
