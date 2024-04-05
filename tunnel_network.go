package guacamole

import (
	"errors"
	"github.com/gorilla/websocket"
	"net"
	"strings"
	"sync"
	"time"
)

// Interface guards
var _ Tunnel = (*NetworkTunnel)(nil)

const connectionTimeout = 5 * time.Second

type NetworkTunnel struct {
	uuid       string         // UUID
	address    string         // guacd address eq: 127.0.0.1:4822
	state      TunnelState    // tunnel state
	conn       net.Conn       // tcp socks
	config     *Configuration // configs
	io         *InstructionIO
	writeMutex sync.Mutex
	recording  *Recording

	debug bool

	closer       chan struct{}
	once         sync.Once
	observers    map[string]Tunnel
	observerLock sync.Mutex

	killer chan string
}

func (t *NetworkTunnel) Kill(reason string) {
	t.killer <- reason
}

func (t *NetworkTunnel) UUID() string {
	return t.uuid
}

func NewNetworkTunnel(address string, config *Configuration, debug bool) Tunnel {
	tun := &NetworkTunnel{
		address:   address,
		config:    config,
		debug:     debug,
		closer:    make(chan struct{}),
		observers: make(map[string]Tunnel),
		killer:    make(chan string, 1),
	}
	return tun
}

func (t *NetworkTunnel) Address() string {
	return t.address
}

func (t *NetworkTunnel) Connect() error {
	conn, err := net.DialTimeout("tcp", t.address, connectionTimeout)
	if err != nil {
		return err
	}

	t.conn = conn
	t.io = NewInstructionIO(t.conn, t.debug)

	config := t.config

	recordingPath := config.GetParameter(RecordingPath)
	if "" != recordingPath {
		config.UnSetParameter(RecordingPath)
		config.UnSetParameter(CreateRecordingPath)
		recording, err := NewRecording(recordingPath)
		if err != nil {
			return err
		}
		t.recording = recording
		go t.recording.Run()
	}

	err = t.handshake()
	if err != nil {
		_ = t.conn.Close()
		return err
	}
	return err
}

func (t *NetworkTunnel) handshake() error {
	config := t.config
	selectArg := config.ConnectionID
	if selectArg == "" {
		selectArg = config.Protocol
	}

	_, err := t.io.Write(NewInstruction("select", selectArg))
	if err != nil {
		return err
	}

	args, err := t.io.Expect("args")
	if err != nil {
		return err
	}

	width := config.GetParameter("width")
	height := config.GetParameter("height")
	dpi := config.GetParameter("dpi")

	// send size
	if _, err := t.io.Write(NewInstruction("size", width, height, dpi)); err != nil {
		return err
	}

	if config.GetParameter(DisableAudio) != "true" {
		if _, err := t.io.Write(NewInstruction("audio", "audio/L8", "audio/L16")); err != nil {
			return err
		}
	}

	if _, err := t.io.Write(NewInstruction("video")); err != nil {
		return err
	}
	if _, err := t.io.Write(NewInstruction("image", "image/jpeg", "image/png", "image/webp")); err != nil {
		return err
	}
	if _, err := t.io.Write(NewInstruction("timezone", "Asia/Shanghai")); err != nil {
		return err
	}

	parameters := make([]string, len(args.Args))
	for i := range args.Args {
		argName := args.Args[i]
		if strings.Contains(argName, "VERSION") {
			parameters[i] = Version
			continue
		}
		parameters[i] = config.GetParameter(argName)
	}
	// send connect
	if _, err := t.io.Write(NewInstruction("connect", parameters...)); err != nil {
		return err
	}

	ready, err := t.io.Expect("ready")
	if err != nil {
		return err
	}

	if len(ready.Args) == 0 {
		return errors.New("no connection id received")
	}

	t.uuid = ready.Args[0]
	t.state = TunnelOpen
	return nil
}

func (t *NetworkTunnel) Send(raw []byte) error {
	if t.state != TunnelOpen {
		return ErrNotConnected
	}

	if len(raw) == 0 {
		return nil
	}

	_, err := t.io.WriteRaw(raw)
	return err
}

func (t *NetworkTunnel) SendInstruction(ins ...*Instruction) error {
	if t.state != TunnelOpen {
		return ErrNotConnected
	}

	if len(ins) == 0 {
		return nil
	}

	t.writeMutex.Lock()
	defer t.writeMutex.Unlock()

	var err error
	for _, in := range ins {
		_, err = t.io.Write(in)
		if err != nil {
			break
		}
	}
	return err
}

func (t *NetworkTunnel) Receive() ([]byte, error) {
	if t.state != TunnelOpen {
		return nil, ErrNotConnected
	}
	p, err := t.io.ReadRaw()
	if err != nil {
		return nil, err
	}
	if t.recording != nil {
		t.recording.Send(p)
	}
	return p, nil
}

func (t *NetworkTunnel) ReceiveInstruction() (*Instruction, error) {
	if t.state != TunnelOpen {
		return nil, ErrNotConnected
	}

	return t.io.Read()
}

func (t *NetworkTunnel) Disconnect() {
	t.once.Do(func() {
		close(t.closer)
		close(t.killer)
		// 发送断开连接的消息
		_ = t.SendInstruction(NewInstruction("disconnect"))
		// 主动断开
		t.closeTunnel()
		// 关闭录制
		if t.recording != nil {
			t.recording.Close()
		}
		t.ClearSharer()
	})
}

func (t *NetworkTunnel) ClearSharer() {
	// 断开观察者
	for _, tunnel := range t.observers {
		tunnel.Disconnect()
	}
}

func (t *NetworkTunnel) closeTunnel() {
	if t.state == TunnelClosed {
		return
	}

	t.state = TunnelClosed
	_ = t.io.Close()
}

func (t *NetworkTunnel) To(ws *websocket.Conn, readonly bool) error {
	go t.loopRead(ws)
	if !readonly {
		go t.loopWrite(ws)
	}
	<-t.closer
	return nil
}

func (t *NetworkTunnel) loopRead(ws *websocket.Conn) {
	defer func() {
		ws.Close()
		t.Disconnect()
	}()
	for {
		select {
		case <-t.closer:
			return
		case reason := <-t.killer:
			_ = ws.WriteMessage(websocket.TextMessage, []byte(NewInstruction("error", reason, "886").String()))
			return
		default:
			instruction, err := t.Receive()
			if err != nil {
				return
			}
			if len(instruction) == 0 {
				continue
			}
			err = ws.WriteMessage(websocket.TextMessage, instruction)
			if err != nil {
				// 前端写入失败
				return
			}
		}
	}
}

func (t *NetworkTunnel) loopWrite(ws *websocket.Conn) {
	defer func() {
		ws.Close()
		t.Disconnect()
	}()
	for {
		select {
		case <-t.closer:
			return
		default:
			for {
				_, message, err := ws.ReadMessage()
				if err != nil {
					// 读取ws失败就代表前端断开了连接，这里要退出
					return
				}
				err = t.Send(message)
				if err != nil {
					return
				}
			}
		}
	}
}

func (t *NetworkTunnel) Join(guest string) Tunnel {
	t.observerLock.Lock()
	defer t.observerLock.Unlock()

	configuration := NewConfiguration()
	configuration.ConnectionID = t.UUID()
	configuration.SetParameter("width", "1024")
	configuration.SetParameter("height", "768")
	configuration.SetParameter("dpi", "96")
	//configuration.SetReadOnlyMode()

	forkedTunnel := NewNetworkTunnel(t.Address(), configuration, false)

	t.observers[guest] = forkedTunnel
	return forkedTunnel
}

func (t *NetworkTunnel) Quit(guest string) {
	t.observerLock.Lock()
	defer t.observerLock.Unlock()

	tunnel, ok := t.observers[guest]
	if ok {
		tunnel.Disconnect()
	}
	delete(t.observers, guest)
}
