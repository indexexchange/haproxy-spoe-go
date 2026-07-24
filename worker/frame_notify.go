package worker

import (
	"fmt"

	"github.com/indexexchange/haproxy-spoe-go/frame"
	"github.com/indexexchange/haproxy-spoe-go/request"
)

func (w *worker) processNotifyFrame(f *frame.Frame) {
	defer frame.ReleaseFrame(f)
	defer w.wg.Done()

	req := request.AcquireRequest()
	defer request.ReleaseRequest(req)

	req.StreamID = f.StreamID
	req.FrameID = f.FrameID
	req.EngineID = w.engineID
	req.Messages = f.Messages

	w.handler(req)

	ackFrame := frame.AcquireFrame()
	defer frame.ReleaseFrame(ackFrame)

	ackFrame.Type = frame.TypeAgentAck
	ackFrame.StreamID = f.StreamID
	ackFrame.FrameID = f.FrameID
	ackFrame.Actions = req.Actions

	err := w.writeFrame(ackFrame)
	if err != nil {
		w.logger.Errorf("ack frame write failed: %v", err)
	}
}

func (w *worker) writeFrame(f *frame.Frame) error {
	// Encode marshals into the frame's pooled buffer and writes it to the
	// connection in a single Write call, which keeps concurrently written
	// frames from interleaving on the wire.
	if _, err := f.Encode(w.conn); err != nil {
		return fmt.Errorf("cannot write frame to connection: %w", err)
	}

	return nil
}
