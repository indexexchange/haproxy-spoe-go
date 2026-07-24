package kv

import (
	"testing"
)

func TestResetKeepsCapacityAndClearsValues(t *testing.T) {
	kv := NewKV()
	for i := 0; i < 16; i++ {
		kv.Add("key", []byte("value"))
	}

	capBefore := cap(kv.m)
	kv.Reset()

	if len(kv.m) != 0 {
		t.Fatalf("expected empty KV after Reset, got len %d", len(kv.m))
	}
	if cap(kv.m) != capBefore {
		t.Fatalf("Reset must keep slice capacity: had %d, got %d", capBefore, cap(kv.m))
	}

	// The backing array must not pin previous values (Item.Value is an
	// interface and would otherwise keep the old data reachable).
	full := kv.m[:cap(kv.m)]
	for i := range full {
		if full[i].Name != "" || full[i].Value != nil {
			t.Fatalf("Reset left stale item at index %d: %+v", i, full[i])
		}
	}
}

func TestResetReuseRoundTrip(t *testing.T) {
	kv := AcquireKV()
	kv.Add("first", int32(1))
	ReleaseKV(kv)

	kv = AcquireKV()
	defer ReleaseKV(kv)

	if _, ok := kv.Get("first"); ok {
		t.Fatal("pooled KV leaked an item from the previous use")
	}

	kv.Add("second", "two")
	v, ok := kv.Get("second")
	if !ok || v != "two" {
		t.Fatalf("expected second=two, got %v (found=%v)", v, ok)
	}
	if len(kv.Data()) != 1 {
		t.Fatalf("expected exactly 1 item, got %d", len(kv.Data()))
	}
}

// Guards the Reset() capacity-keeping behavior. In production profiles
// (at 300k QPS) the old Reset — kv.m = make([]Item, 0) —
// forced every pooled KV to re-grow its slice from zero per message,
// accounting for ~24% of all allocated bytes in the process.
//
// Reset+16 Adds, Intel Ultra 7 265HX, go1.26:
//
//	before (make([]Item, 0)):    243.3 ns/op   992 B/op   5 allocs/op
//	after  (clear + kv.m[:0]):    27.4 ns/op     0 B/op   0 allocs/op
func BenchmarkAddAfterReset(b *testing.B) {
	kv := NewKV()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kv.Reset()
		for j := 0; j < 16; j++ {
			kv.Add("key", int64(j))
		}
	}
}
