package guacenc

import (
	"image"
)

// OnSyncFunc is the signature for OnSync event handlers. It will receive the current screen image and the
// timestamp of the last update.
type OnSyncFunc = func(image image.Image, lastUpdate int64)

// Client is the main struct in this library, it represents the Guacamole protocol client.
// Automatically handles incoming and outgoing Guacamole instructions, updating its display
// using one or more graphic primitives.
type Client struct {
	session Session
	display *display
	streams streams
	logger  Logger
	onSync  OnSyncFunc
}

func NewRecordingClient(path string, logger ...Logger) (*Client, error) {
	var log Logger
	if len(logger) > 0 {
		log = logger[0]
	} else {
		log = &DefaultLogger{}
	}

	s, err := newFileSession(path, log)
	if err != nil {
		return nil, err
	}

	c := &Client{
		session: s,
		display: newDisplay(log),
		streams: newStreams(),
		logger:  log,
	}
	return c, nil
}

// Start the Client's main loop. It is a blocking call, so it
// should be called in its on goroutine
func (c *Client) Start() {
	for {
		ch := c.session.Read()
		select {
		case ins, alive := <-ch:
			if !alive {
				return
			}
			h, ok := handlers[ins.Opcode]
			if !ok {
				c.logger.Errorf("Instruction not implemented: %s", ins.Opcode)
				continue
			}
			err := h(c, ins.Args)
			if err != nil {
				c.session.Terminate()
			}
		}
	}
}

// OnSync sets a function that will be called on every sync instruction received. This event
// usually happens after a batch of updates are received from the guacd server, making it a
// perfect way to get the current screenshot without having to poll with Screen().
// The handler is expected to be called frequently, so avoid adding any blocking behaviour.
// If your handler is slow, consider using a concurrent pattern (using goroutines)
func (c *Client) OnSync(f OnSyncFunc) {
	c.onSync = f
}

// Screen returns a snapshot of the current screen, together with the last updated timestamp
func (c *Client) Screen() (image image.Image, lastUpdate int64) {
	return c.display.getCanvas()
}

// State returns the current session state
func (c *Client) State() SessionState {
	return c.session.State()
}
