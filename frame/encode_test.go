package frame

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/indexexchange/haproxy-spoe-go/action"
)

func TestFrame_Write(t *testing.T) {
	f := NewFrame()
	f.Type = TypeAgentDisconnect
	f.FrameID = 123
	f.StreamID = 456
	f.KV.Add("key1", "val1")
	f.KV.Add("key2", "val2")
	buf := &bytes.Buffer{}
	frameSize, err := f.Encode(buf)
	if err != nil {
		t.Fatalf("expect err is nil, got %v", err)
	}
	bufBytes := buf.Bytes()
	encodedFrameSize := int(binary.BigEndian.Uint32(bufBytes[0:4]))
	if frameSize-4 != encodedFrameSize {
		t.Fatal("wrong frame size")
	}
	if string(bufBytes[13:17]) != "key1" {
		t.Fatal("expect key1")
	}
	if string(bufBytes[19:23]) != "val1" {
		t.Fatal("expect val1")
	}
	if string(bufBytes[24:28]) != "key2" {
		t.Fatal("expect key2")
	}
	if string(bufBytes[30:34]) != "val2" {
		t.Fatal("expect val1")
	}
}

// Encode reuses f.writeBuf across pool cycles; a longer first encode must
// not leak stale bytes into a shorter second one, and both must go to dest
// as a single Write (concurrent ACK writers share one connection).
func TestFrame_EncodeReuse(t *testing.T) {
	f := AcquireFrame()
	defer ReleaseFrame(f)

	f.Type = TypeAgentDisconnect
	f.FrameID = 1
	f.StreamID = 1
	f.KV.Add("key-that-is-long-enough", "value-that-is-long-enough-to-matter")

	first := &bytes.Buffer{}
	if _, err := f.Encode(first); err != nil {
		t.Fatalf("first encode: %v", err)
	}

	f.Reset()
	f.Type = TypeAgentDisconnect
	f.FrameID = 2
	f.StreamID = 2
	f.KV.Add("k", "v")

	second := &bytes.Buffer{}
	n, err := f.Encode(second)
	if err != nil {
		t.Fatalf("second encode: %v", err)
	}
	if n != second.Len() {
		t.Fatalf("Encode reported %d bytes, wrote %d", n, second.Len())
	}
	if second.Len() >= first.Len() {
		t.Fatalf("second frame should be shorter: first %d, second %d", first.Len(), second.Len())
	}

	got := second.Bytes()
	encodedFrameSize := int(binary.BigEndian.Uint32(got[0:4]))
	if encodedFrameSize != second.Len()-4 {
		t.Fatalf("stale bytes leaked: length prefix %d, payload %d", encodedFrameSize, second.Len()-4)
	}

	// Byte-identical to an encode from a fresh frame.
	fresh := NewFrame()
	fresh.Type = TypeAgentDisconnect
	fresh.FrameID = 2
	fresh.StreamID = 2
	fresh.KV.Add("k", "v")
	want := &bytes.Buffer{}
	if _, err := fresh.Encode(want); err != nil {
		t.Fatalf("fresh encode: %v", err)
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Fatalf("reused-frame encode differs from fresh encode:\ngot  %x\nwant %x", got, want.Bytes())
	}
}

func BenchmarkFrame_Encode(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f := NewFrame()
		f.Type = TypeAgentDisconnect
		f.FrameID = 123
		f.StreamID = 456
		f.KV.Add("key1", "val1")
		f.KV.Add("key2", "val2")
		buf := &bytes.Buffer{}
		_, _ = f.Encode(buf)
	}
}

// BenchmarkFrame_EncodeAck mirrors the production ACK write path: a pooled
// frame carrying SetVar actions, encoded per response. Encoding into the
// frame's retained writeBuf was added because this path was ~46% of all
// allocated bytes in L7 production profiles at 300k QPS (2026-07-23):
// Action.Marshal grew a nil slice per ACK (27.2%) and Encode built a fresh
// bytes.Buffer per frame which writeFrame then copied into a second one
// (18.4%).
//
// 4 SetVar actions, Intel Ultra 7 265HX, go1.26 (Encode only — production
// additionally saved writeFrame's second copy into a bytes.NewBuffer):
//
//	before (buffer per frame): 272.8 ns/op   488 B/op   11 allocs/op
//	after  (pooled writeBuf):   69.9 ns/op     0 B/op    0 allocs/op
func BenchmarkFrame_EncodeAck(b *testing.B) {
	var actions action.Actions
	actions.SetVar(action.ScopeTransaction, "ip_score", int64(42))
	actions.SetVar(action.ScopeTransaction, "verdict", "allow")
	actions.SetVar(action.ScopeTransaction, "rule_id", "some-rule-identifier")
	actions.SetVar(action.ScopeTransaction, "sampled", true)

	f := AcquireFrame()
	defer ReleaseFrame(f)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Reset()
		f.Type = TypeAgentAck
		f.StreamID = uint64(i)
		f.FrameID = uint64(i)
		f.Actions = actions

		if _, err := f.Encode(io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}
