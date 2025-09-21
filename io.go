package guacamole

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Protocol constants matching Java implementation
const (
	InstructionMaxLength   = 8192 // Maximum number of characters per instruction
	InstructionMaxDigits   = 5    // Maximum number of digits per length prefix
	InstructionMaxElements = 128  // Maximum number of elements per instruction
)

var (
	ErrInstructionParseFailed = errors.New("instruction parse failed")
	ErrInstructionTooLong     = errors.New("instruction exceeds maximum length")
	ErrTooManyElements        = errors.New("instruction contains too many elements")
	ErrInvalidLengthPrefix    = errors.New("non-numeric character in element length")
	ErrInvalidTerminator      = errors.New("element terminator was not ';' nor ','")
)

// InstructionIO ...
type InstructionIO struct {
	conn   io.ReadWriteCloser
	reader *bufio.Reader
	writer *bufio.Writer

	debug bool
}

// NewInstructionIO ...
func NewInstructionIO(conn io.ReadWriteCloser, debug bool) *InstructionIO {
	return &InstructionIO{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
		debug:  debug,
	}
}

// Close closes the InstructionIO
func (io *InstructionIO) Close() error {
	return io.conn.Close()
}

// ReadRaw reads raw data from io reader
func (io *InstructionIO) ReadRaw() (p []byte, err error) {
	data, err := io.reader.ReadBytes(byte(Delimiter))
	if err != nil {
		return nil, err
	}
	s := string(data)
	if io.debug {
		println("<- ", s)
	}
	if s == "rate=44100,channels=2;" {
		return make([]byte, 0), nil
	}
	if s == "rate=22050,channels=2;" {
		return make([]byte, 0), nil
	}
	if s == "5.audio,1.1,31.audio/L16;" {
		s += "rate=44100,channels=2;"
	}
	return []byte(s), nil
}

// Read reads and parses the instruction from io reader
func (io *InstructionIO) Read() (*Instruction, error) {
	raw, err := io.ReadRaw()
	if err != nil {
		return nil, err
	}
	return ParseInstruction(raw)
}

// WriteRaw writes raw buffer into io writer
func (io *InstructionIO) WriteRaw(buf []byte) (n int, err error) {
	n, err = io.writer.Write(buf)
	if err != nil {
		return
	}
	if io.debug {
		println("-> ", string(buf))
	}
	err = io.writer.Flush()
	return
}

// Write writes and decodes an instruction to io writer
func (io *InstructionIO) Write(ins *Instruction) (int, error) {
	return io.WriteRaw([]byte(ins.String()))
}

func (io *InstructionIO) Expect(opcode string) (*Instruction, error) {
	instruction, err := io.Read()
	if err != nil {
		return nil, err
	}

	if opcode != instruction.Opcode {
		msg := fmt.Sprintf(`expected "%s" instruction but instead received "%s:%s"`, opcode, instruction.Opcode, instruction.String())
		return instruction, errors.New(msg)
	}
	return instruction, nil
}

func ParseInstruction(raw []byte) (*Instruction, error) {
	if len(raw) == 0 {
		return NewInstruction("nop"), nil
	}

	content := string(raw)

	// Handle special audio cases (legacy compatibility)
	if content == "5.audio,1.1,31.audio/L16;rate=44100,channels=2;" {
		return NewInstruction("audio", "1", "audio/L16;rate=44100,channels=2"), nil
	}
	if content == "5.audio,1.0,31.audio/L16;rate=44100,channels=2;" {
		return NewInstruction("audio", "0", "audio/L16;rate=44100,channels=2"), nil
	}

	// Remove trailing semicolon
	if strings.HasSuffix(content, ";") {
		content = strings.TrimSuffix(content, ";")
	}

	// Split by comma to get elements
	messages := strings.Split(content, ",")
	if len(messages) > InstructionMaxElements {
		return nil, ErrTooManyElements
	}

	args := make([]string, len(messages))
	for i, message := range messages {
		// Parse length.content format
		parts := strings.SplitN(message, ".", 2)
		if len(parts) < 2 {
			return nil, ErrInstructionParseFailed
		}

		// Validate length prefix (should be numeric)
		lengthStr := parts[0]
		if len(lengthStr) > InstructionMaxDigits {
			return nil, ErrInvalidLengthPrefix
		}

		for _, char := range lengthStr {
			if char < '0' || char > '9' {
				return nil, ErrInvalidLengthPrefix
			}
		}

		args[i] = parts[1]
	}

	if len(args) == 1 {
		return NewInstruction(args[0]), nil
	}
	return NewInstruction(args[0], args[1:]...), nil
}
