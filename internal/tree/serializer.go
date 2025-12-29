package tree

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// enc is the byte order used for all serialization.
// BigEndian ensures consistent hashing across platforms.
var enc = binary.BigEndian

// Serializer converts typed values to bytes for consistent hashing.
// All readers should use this to ensure identical data produces identical hashes.
type Serializer struct {
	buf bytes.Buffer
}

// NewSerializer creates a new Serializer.
func NewSerializer() *Serializer {
	return &Serializer{}
}

// SerializeRow converts a row's values to bytes for hashing.
// The format is deterministic: each value is type-tagged and length-prefixed.
func (s *Serializer) SerializeRow(values []any) []byte {
	s.buf.Reset()

	for _, v := range values {
		s.serializeValue(v)
	}

	// Return a copy to avoid buffer reuse issues
	result := make([]byte, s.buf.Len())
	copy(result, s.buf.Bytes())
	return result
}

// serializeValue writes a single value to the buffer.
func (s *Serializer) serializeValue(v any) {
	switch val := v.(type) {
	case nil:
		s.writeType(0)

	case string:
		s.writeType(1)
		s.writeBytes([]byte(val))

	case int:
		s.writeType(2)
		s.writeInt64(int64(val))

	case int64:
		s.writeType(2)
		s.writeInt64(val)

	case int32:
		s.writeType(2)
		s.writeInt64(int64(val))

	case float64:
		s.writeType(3)
		s.writeFloat64(val)

	case float32:
		s.writeType(3)
		s.writeFloat64(float64(val))

	case bool:
		s.writeType(4)
		if val {
			s.buf.WriteByte(1)
		} else {
			s.buf.WriteByte(0)
		}

	case []byte:
		s.writeType(5)
		s.writeBytes(val)

	case time.Time:
		s.writeType(6)
		s.writeInt64(val.UnixNano())

	default:
		// Fallback: convert to string
		s.writeType(1)
		s.writeBytes([]byte(fmt.Sprintf("%v", val)))
	}
}

func (s *Serializer) writeType(t byte) {
	s.buf.WriteByte(t)
}

func (s *Serializer) writeBytes(b []byte) {
	// Length prefix (4 bytes)
	lenBuf := make([]byte, 4)
	enc.PutUint32(lenBuf, uint32(len(b)))
	s.buf.Write(lenBuf)
	s.buf.Write(b)
}

func (s *Serializer) writeInt64(i int64) {
	buf := make([]byte, 8)
	enc.PutUint64(buf, uint64(i))
	s.buf.Write(buf)
}

func (s *Serializer) writeFloat64(f float64) {
	buf := make([]byte, 8)
	enc.PutUint64(buf, math.Float64bits(f))
	s.buf.Write(buf)
}
