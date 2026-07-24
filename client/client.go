package client

import (
	"bufio"
	"fmt"
	"io"
	"net"

	"github.com/indexexchange/haproxy-spoe-go/frame"
)

// Client is a simple client for spop protocol, this should only be used for testing purpose
type Client struct {
	conn   net.Conn
	reader io.Reader
}

// NewClient create a new Client for an established connection
func NewClient(conn net.Conn) Client {
	return Client{conn: conn, reader: bufio.NewReader(conn)}
}

// Init initialize the client by sending the HaproxyHello frame
func (c *Client) Init() error {
	f := frame.AcquireFrame()
	defer frame.ReleaseFrame(f)
	f.Type = frame.TypeHaproxyHello
	f.StreamID = 0
	f.FrameID = 0
	f.KV.Add("supported-versions", "2")
	f.KV.Add("max-frame-size", uint32(16*1024))
	f.KV.Add("capabilities", "")

	err := c.send(f)
	if err != nil {
		return err
	}

	responseFrame := frame.AcquireFrame()
	defer frame.ReleaseFrame(responseFrame)
	responseFrame.Read(c.reader)

	switch responseFrame.Type {
	case frame.TypeAgentHello:
		if responseFrame.FrameID != uint64(0) || responseFrame.StreamID != uint64(0) {
			return fmt.Errorf("FrameID or StreamID mismatch")
		}
	default:
		return fmt.Errorf("unexpected frame type: %v", responseFrame.Type)
	}

	return nil

}

func (c *Client) send(f *frame.Frame) error {
	_, err := f.Encode(c.conn)
	return err
}

// Notify send an empty Notify frame
func (c *Client) Notify() error {
	f := frame.AcquireFrame()
	defer frame.ReleaseFrame(f)
	f.Type = frame.TypeNotify
	f.StreamID = 1
	f.FrameID = 1

	err := c.send(f)
	if err != nil {
		return err
	}

	responseFrame := frame.AcquireFrame()
	defer frame.ReleaseFrame(responseFrame)
	responseFrame.Read(c.reader)
	return nil
}

// Stop the client by sending HaproxyDisconnect frame
func (c *Client) Stop() error {
	f := frame.AcquireFrame()
	defer frame.ReleaseFrame(f)
	f.Type = frame.TypeHaproxyDisconnect
	f.StreamID = 0
	f.FrameID = 0
	f.KV.Add("status-code", uint32(0))
	f.KV.Add("message", "normal")

	err := c.send(f)
	if err != nil {
		return err
	}

	responseFrame := frame.AcquireFrame()
	defer frame.ReleaseFrame(responseFrame)
	responseFrame.Read(c.reader)

	return nil

}
