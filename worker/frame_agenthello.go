package worker

import (
	"fmt"

	"github.com/indexexchange/haproxy-spoe-go/frame"
)

func (w *worker) sendAgentHello(haproxyHello *frame.Frame) error {
	agentHello := frame.AcquireFrame()
	defer frame.ReleaseFrame(agentHello)

	agentHello.Type = frame.TypeAgentHello
	agentHello.FrameID = haproxyHello.FrameID
	agentHello.StreamID = haproxyHello.StreamID

	agentHello.KV.Add("version", "2.0")
	agentHello.KV.Add("max-frame-size", haproxyHello.MaxFrameSize)
	agentHello.KV.Add("capabilities", capabilities)

	if _, err := agentHello.Encode(w.conn); err != nil {
		return fmt.Errorf("error write to connection: %v", err)
	}

	return nil
}
