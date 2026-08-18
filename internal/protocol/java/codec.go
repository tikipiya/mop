package java

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	DefaultMaxPacketBytes = 1 << 20
	DefaultMaxStringBytes = 1 << 20
	maxVarIntBytes        = 5
)

var (
	ErrVarIntTooLong    = errors.New("varint exceeds 5 bytes")
	ErrVarIntOutOfRange = errors.New("varint exceeds 32 bits")
	ErrNegativeLength   = errors.New("negative length")
	ErrEmptyPacket      = errors.New("packet has no packet id")
	ErrPacketTooLarge   = errors.New("packet exceeds configured limit")
	ErrStringTooLarge   = errors.New("string exceeds configured limit")
	ErrInvalidUTF8      = errors.New("string is not valid UTF-8")
	ErrUnexpectedPacket = errors.New("unexpected packet id")
)

type Codec struct {
	MaxPacketBytes int32
	MaxStringBytes int32
}

func NewCodec() Codec {
	return Codec{
		MaxPacketBytes: DefaultMaxPacketBytes,
		MaxStringBytes: DefaultMaxStringBytes,
	}
}

func EncodeVarInt(value int32) []byte {
	u := uint32(value)
	out := make([]byte, 0, maxVarIntBytes)
	for {
		if u&^uint32(0x7f) == 0 {
			return append(out, byte(u))
		}
		out = append(out, byte(u&0x7f)|0x80)
		u >>= 7
	}
}

func DecodeVarInt(r io.ByteReader) (int32, error) {
	var value uint32
	for i := 0; i < maxVarIntBytes; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		if i == maxVarIntBytes-1 {
			if b&0x80 != 0 {
				return 0, ErrVarIntTooLong
			}
			if b&0x70 != 0 {
				return 0, ErrVarIntOutOfRange
			}
		}
		value |= uint32(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return int32(value), nil
		}
	}
	return 0, ErrVarIntTooLong
}

func (c Codec) WriteString(w io.Writer, value string) error {
	if !utf8.ValidString(value) {
		return ErrInvalidUTF8
	}
	limit := c.stringLimit()
	if int64(len(value)) > int64(limit) {
		return ErrStringTooLarge
	}
	if _, err := w.Write(EncodeVarInt(int32(len(value)))); err != nil {
		return err
	}
	_, err := io.WriteString(w, value)
	return err
}

func (c Codec) ReadString(r io.Reader) (string, error) {
	length, err := DecodeVarInt(byteReader{r: r})
	if err != nil {
		return "", err
	}
	if length < 0 {
		return "", ErrNegativeLength
	}
	if length > c.stringLimit() {
		return "", ErrStringTooLarge
	}
	data := make([]byte, int(length))
	if _, err := io.ReadFull(r, data); err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", ErrInvalidUTF8
	}
	return string(data), nil
}

func (c Codec) WritePacket(w io.Writer, packetID int32, payload []byte) error {
	id := EncodeVarInt(packetID)
	length := len(id) + len(payload)
	if length > int(c.packetLimit()) {
		return ErrPacketTooLarge
	}
	if _, err := w.Write(EncodeVarInt(int32(length))); err != nil {
		return err
	}
	if _, err := w.Write(id); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func (c Codec) ReadPacket(r io.Reader, expectedID int32) ([]byte, error) {
	length, err := DecodeVarInt(byteReader{r: r})
	if err != nil {
		return nil, err
	}
	if length < 0 {
		return nil, ErrNegativeLength
	}
	if length == 0 {
		return nil, ErrEmptyPacket
	}
	if length > c.packetLimit() {
		return nil, ErrPacketTooLarge
	}

	body := make([]byte, int(length))
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	bodyReader := bytes.NewReader(body)
	packetID, err := DecodeVarInt(bodyReader)
	if err != nil {
		return nil, fmt.Errorf("decode packet id: %w", err)
	}
	if packetID != expectedID {
		return nil, fmt.Errorf("%w: got 0x%x, want 0x%x", ErrUnexpectedPacket, packetID, expectedID)
	}
	payload := make([]byte, bodyReader.Len())
	copy(payload, body[len(body)-bodyReader.Len():])
	return payload, nil
}

func WriteInt64(w io.Writer, value int64) error {
	return binary.Write(w, binary.BigEndian, value)
}

func ReadInt64(r io.Reader) (int64, error) {
	var value int64
	err := binary.Read(r, binary.BigEndian, &value)
	return value, err
}

func WriteUint16(w io.Writer, value uint16) error {
	return binary.Write(w, binary.BigEndian, value)
}

func ReadUint16(r io.Reader) (uint16, error) {
	var value uint16
	err := binary.Read(r, binary.BigEndian, &value)
	return value, err
}

func (c Codec) packetLimit() int32 {
	if c.MaxPacketBytes <= 0 {
		return DefaultMaxPacketBytes
	}
	return c.MaxPacketBytes
}

func (c Codec) stringLimit() int32 {
	if c.MaxStringBytes <= 0 {
		return DefaultMaxStringBytes
	}
	return c.MaxStringBytes
}

type byteReader struct{ r io.Reader }

func (r byteReader) ReadByte() (byte, error) {
	var one [1]byte
	_, err := io.ReadFull(r.r, one[:])
	return one[0], err
}
