package netutil

import (
	"encoding/binary"
	"errors"
)

// MsgType identifies the type of the gossip message.
type MsgType uint8

const (
	MsgDelta MsgType = iota
)

// Delta represents a rate limiter token consumption delta.
// It tracks how many tokens a specific client consumed.
type Delta struct {
	ClientID string
	Consumed uint64
}

// EncodeDeltaMessage zero-alloc encodes a slice of Deltas into the provided buffer.
// The format is:
// 1 byte: MsgType
// 4 bytes: Sequence ID
// 2 bytes: Number of deltas
// For each delta:
//   2 bytes: ClientID string length
//   N bytes: ClientID string
//   8 bytes: Consumed tokens count
// Returns the number of bytes written, or an error if the buffer is too small.
func EncodeDeltaMessage(buf []byte, seqID uint32, deltas []Delta) (int, error) {
	if len(buf) < 7 { // type(1) + seq(4) + numDeltas(2)
		return 0, errors.New("buffer too small")
	}

	buf[0] = byte(MsgDelta)
	binary.BigEndian.PutUint32(buf[1:5], seqID)
	binary.BigEndian.PutUint16(buf[5:7], uint16(len(deltas)))

	offset := 7
	for _, d := range deltas {
		clientLen := len(d.ClientID)

		// 2 bytes length, client string, 8 bytes consumed
		reqSpace := 2 + clientLen + 8
		if offset+reqSpace > len(buf) {
			return 0, errors.New("buffer too small for deltas")
		}

		binary.BigEndian.PutUint16(buf[offset:], uint16(clientLen))
		offset += 2

		copy(buf[offset:], d.ClientID)
		offset += clientLen

		binary.BigEndian.PutUint64(buf[offset:], d.Consumed)
		offset += 8
	}

	return offset, nil
}

// DecodeDeltaMessage decodes a delta message from a byte slice.
func DecodeDeltaMessage(buf []byte) (uint32, []Delta, error) {
	if len(buf) < 7 {
		return 0, nil, errors.New("invalid buffer length")
	}
	if MsgType(buf[0]) != MsgDelta {
		return 0, nil, errors.New("not a delta message")
	}

	seqID := binary.BigEndian.Uint32(buf[1:5])
	numDeltas := int(binary.BigEndian.Uint16(buf[5:7]))

	deltas := make([]Delta, 0, numDeltas)
	offset := 7

	for i := 0; i < numDeltas; i++ {
		if offset+2 > len(buf) {
			return 0, nil, errors.New("invalid buffer: missing client length")
		}
		clientLen := int(binary.BigEndian.Uint16(buf[offset:]))
		offset += 2

		if offset+clientLen+8 > len(buf) {
			return 0, nil, errors.New("invalid buffer: missing client ID or consumed count")
		}
		clientID := string(buf[offset : offset+clientLen])
		offset += clientLen

		consumed := binary.BigEndian.Uint64(buf[offset:])
		offset += 8

		deltas = append(deltas, Delta{
			ClientID: clientID,
			Consumed: consumed,
		})
	}

	return seqID, deltas, nil
}
