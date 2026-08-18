package java

import (
	"bytes"
	"errors"
	"io"
	"math"
	"strconv"
	"testing"
)

func TestVarIntRoundTrip(t *testing.T) {
	t.Parallel()
	values := []int32{0, 1, 127, 128, 255, math.MaxInt32, -1, math.MinInt32}
	for _, value := range values {
		value := value
		t.Run(strconv.FormatInt(int64(value), 10), func(t *testing.T) {
			encoded := EncodeVarInt(value)
			if len(encoded) > maxVarIntBytes {
				t.Fatalf("encoded length = %d", len(encoded))
			}
			got, err := DecodeVarInt(bytes.NewReader(encoded))
			if err != nil {
				t.Fatalf("DecodeVarInt: %v", err)
			}
			if got != value {
				t.Fatalf("round trip = %d, want %d", got, value)
			}
		})
	}
}

func TestDecodeVarIntRejectsInvalidEncoding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data []byte
		want error
	}{
		{"too long", []byte{0x80, 0x80, 0x80, 0x80, 0x80}, ErrVarIntTooLong},
		{"out of range", []byte{0x80, 0x80, 0x80, 0x80, 0x10}, ErrVarIntOutOfRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeVarInt(bytes.NewReader(tt.data))
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestStringRoundTripAndLimits(t *testing.T) {
	t.Parallel()
	codec := Codec{MaxStringBytes: 32}
	var encoded bytes.Buffer
	const want = "Minecraft 日本語"
	if err := codec.WriteString(&encoded, want); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	got, err := codec.ReadString(&encoded)
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	if got != want {
		t.Fatalf("ReadString = %q, want %q", got, want)
	}

	codec.MaxStringBytes = 3
	if err := codec.WriteString(io.Discard, "four"); !errors.Is(err, ErrStringTooLarge) {
		t.Fatalf("oversized write error = %v", err)
	}
	if _, err := codec.ReadString(bytes.NewReader([]byte{4, 'f', 'o', 'u', 'r'})); !errors.Is(err, ErrStringTooLarge) {
		t.Fatalf("oversized read error = %v", err)
	}
}

func TestPacketRoundTripWithFragmentedReads(t *testing.T) {
	t.Parallel()
	codec := NewCodec()
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	var encoded bytes.Buffer
	if err := codec.WritePacket(&encoded, 0x01, payload); err != nil {
		t.Fatalf("WritePacket: %v", err)
	}

	got, err := codec.ReadPacket(&oneByteReader{data: encoded.Bytes()}, 0x01)
	if err != nil {
		t.Fatalf("ReadPacket: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %v, want %v", got, payload)
	}
}

func TestReadPacketRejectsBadBoundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		codec Codec
		data  []byte
		want  error
	}{
		{"negative length", NewCodec(), EncodeVarInt(-1), ErrNegativeLength},
		{"empty packet", NewCodec(), []byte{0}, ErrEmptyPacket},
		{"too large before allocation", Codec{MaxPacketBytes: 2}, []byte{3}, ErrPacketTooLarge},
		{"short body", NewCodec(), []byte{2, 0}, io.ErrUnexpectedEOF},
		{"wrong id", NewCodec(), []byte{1, 2}, ErrUnexpectedPacket},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.codec.ReadPacket(bytes.NewReader(tt.data), 0)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestInt64BigEndianRoundTrip(t *testing.T) {
	t.Parallel()
	const want int64 = -0x102030405060708
	var encoded bytes.Buffer
	if err := WriteInt64(&encoded, want); err != nil {
		t.Fatalf("WriteInt64: %v", err)
	}
	got, err := ReadInt64(&encoded)
	if err != nil {
		t.Fatalf("ReadInt64: %v", err)
	}
	if got != want {
		t.Fatalf("ReadInt64 = %d, want %d", got, want)
	}
}

type oneByteReader struct {
	data []byte
	pos  int
}

func (r *oneByteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}
