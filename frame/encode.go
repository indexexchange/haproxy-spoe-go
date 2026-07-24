package frame

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/indexexchange/haproxy-spoe-go/varint"
)

// Encode marshals the frame and writes it to dest.
//
// The whole wire frame (4-byte length prefix included) is built in
// f.writeBuf, which is retained across pool cycles, and written to dest
// in a single Write call. The single Write matters for correctness, not
// just speed: ACK frames are encoded onto a shared connection from
// concurrent NOTIFY handler goroutines, and one Write per frame is what
// keeps frames from interleaving on the wire.
func (f *Frame) Encode(dest io.Writer) (int, error) {
	buf := append(f.writeBuf[:0], 0, 0, 0, 0) // frame length, filled in below

	buf = append(buf, byte(f.Type))

	binary.BigEndian.PutUint32(f.tmp[:], f.Flags)
	buf = append(buf, f.tmp[0:4]...)

	n := varint.PutUvarint(f.varintBuf[:], f.StreamID)
	buf = append(buf, f.varintBuf[:n]...)

	n = varint.PutUvarint(f.varintBuf[:], f.FrameID)
	buf = append(buf, f.varintBuf[:n]...)

	switch f.Type {
	case TypeAgentHello, TypeAgentDisconnect, TypeHaproxyHello, TypeHaproxyDisconnect:
		payload, err := f.KV.Bytes()
		if err != nil {
			return 0, err
		}
		buf = append(buf, payload...)

	case TypeAgentAck:
		for _, act := range f.Actions {
			var err error
			buf, err = act.Marshal(buf)
			if err != nil {
				return 0, err
			}
		}

	case TypeNotify:
		if len(*f.Messages) > 0 {
			return 0, fmt.Errorf("encoding Notify frame with Message isn't handled yet")
		}
	default:
		return 0, fmt.Errorf("unexpected frame type %d", f.Type)
	}

	binary.BigEndian.PutUint32(buf[0:4], uint32(len(buf)-4))
	f.writeBuf = buf

	n, err := dest.Write(buf)
	if err != nil || n != len(buf) {
		return 0, fmt.Errorf("error write frame. writes %d, expect %d, err: %v", n, len(buf), err)
	}

	return n, nil
}
