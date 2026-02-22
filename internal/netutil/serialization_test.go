package netutil

import (
	"reflect"
	"testing"
)

func TestEncodeDecodeDeltaMessage(t *testing.T) {
	deltas := []Delta{
		{ClientID: "client1", Consumed: 5},
		{ClientID: "client-long-id-12345", Consumed: 100},
	}

	buf := make([]byte, 1024)
	n, err := EncodeDeltaMessage(buf, 42, deltas)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}

	seqID, decodedDeltas, err := DecodeDeltaMessage(buf[:n])
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if seqID != 42 {
		t.Errorf("expected sequence ID 42, got %d", seqID)
	}

	if !reflect.DeepEqual(deltas, decodedDeltas) {
		t.Errorf("expected deltas %+v, got %+v", deltas, decodedDeltas)
	}
}

func BenchmarkEncodeDeltaMessage(b *testing.B) {
	deltas := []Delta{
		{ClientID: "client1", Consumed: 5},
		{ClientID: "client2", Consumed: 10},
		{ClientID: "client3", Consumed: 15},
	}
	buf := make([]byte, 1024)
	
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeDeltaMessage(buf, uint32(i), deltas)
	}
}

func BenchmarkDecodeDeltaMessage(b *testing.B) {
	deltas := []Delta{
		{ClientID: "client1", Consumed: 5},
		{ClientID: "client2", Consumed: 10},
		{ClientID: "client3", Consumed: 15},
	}
	buf := make([]byte, 1024)
	n, _ := EncodeDeltaMessage(buf, 42, deltas)
	msgBuf := buf[:n]
	
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = DecodeDeltaMessage(msgBuf)
	}
}
