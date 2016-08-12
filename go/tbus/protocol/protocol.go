package protocol

import (
	"bytes"
	"fmt"
	"io"
)

// Prefix and Flag
const (
	PfxRoutingMask    uint8 = 0xe0
	PfxRouting        uint8 = 0xc0
	PfxRoutingAddrNum uint8 = 0x1f

	RoutingAddrsMax = 32

	FormatMask uint8 = 0xf0
	Format     uint8 = 0x10 // rev 1, protobuf encoded

	BodyError uint8 = 0x80 // body contains error
)

// MsgHead is message head
type MsgHead struct {
	Prefix    uint8
	Addrs     []uint8
	Flag      uint8
	MsgID     uint32
	BodyBytes uint16
	Raw       []byte
	RawPrefix []byte
	RawHead   []byte
}

// NeedRoute indicates the message contains routing addresses
func (h *MsgHead) NeedRoute() bool {
	return (h.Prefix&PfxRoutingMask) == PfxRouting && len(h.Addrs) > 0
}

// MsgBody is message body
type MsgBody struct {
	Flag uint8
	Data []byte
	Raw  []byte
}

// Msg is a completely message
type Msg struct {
	Head MsgHead
	Body MsgBody
}

// DecodeHead decodes message head from input
func DecodeHead(reader io.Reader) (head MsgHead, err error) {
	var buf bytes.Buffer
	if _, err = io.CopyN(&buf, reader, 1); err != nil {
		return
	}
	pfx := uint8(buf.Bytes()[0])
	var addrNum int
	if (pfx & PfxRoutingMask) == PfxRouting {
		head.Prefix = pfx
		addrNum = int((pfx & PfxRoutingAddrNum) + 1)
		if _, err = io.CopyN(&buf, reader, int64(addrNum)); err != nil {
			return
		}
	} else if (pfx & FormatMask) == Format {
		head.Flag = pfx
		if head.MsgID, err = decode7bit32(&buf, reader); err != nil {
			return
		}
		if head.BodyBytes, err = decode7bit16(&buf, reader); err != nil {
			return
		}
	} else {
		err = fmt.Errorf("unknown byte 0x%02x", pfx)
		return
	}

	head.Raw = buf.Bytes()
	if addrNum > 0 {
		head.Addrs = head.Raw[1 : addrNum+1]
		head.RawPrefix = head.Raw[0 : addrNum+1]
		head.RawHead = head.Raw[addrNum+1:]
	} else {
		head.RawHead = head.Raw
	}
	return
}

// DecodeBody decodes message body from input
func DecodeBody(reader io.Reader, bodyBytes uint16) (body MsgBody, err error) {
	body.Raw = make([]byte, bodyBytes)
	if _, err = io.ReadFull(reader, body.Raw); err != nil {
		return
	}
	if bodyBytes > 0 {
		body.Flag = uint8(body.Raw[0])
		body.Data = body.Raw[1:]
	}
	return
}

// Decode decodes a complete message
func Decode(reader io.Reader) (msg Msg, err error) {
	if msg.Head, err = DecodeHead(reader); err != nil {
		return
	}
	msg.Body, err = DecodeBody(reader, msg.Head.BodyBytes)
	return
}

// DecodeError returns error if the message contains error
func DecodeError(msg *Msg) error {
	// TODO
	return nil
}

// EncodeRoute encodes routing prefix
func EncodeRoute(w io.Writer, addrs []uint8) (n int, err error) {
	if l := len(addrs); l > RoutingAddrsMax {
		panic("number of addresses exceeds limit")
	} else if l > 0 {
		pfx := []byte{PfxRouting | uint8(l-1)}
		if _, err = w.Write(pfx); err != nil {
			return
		}
		if n, err = w.Write(addrs); err != nil {
			n = 1
			return
		}
		n++
	}
	return
}

// EncodeMsg encodes a message
func EncodeMsg(w io.Writer, msgID uint32, bodyFlag uint8, body []byte) (n int, err error) {
	b := []byte{Format}
	if _, err = w.Write(b); err != nil {
		return
	}
	n++

	var c int
	if c, err = encode7bit(w, uint64(msgID)); err != nil {
		return
	}
	n += c
	if c, err = encode7bit(w, uint64(len(body)+1)); err != nil {
		return
	}
	n += c
	b[0] = bodyFlag
	if _, err = w.Write(b); err != nil {
		return
	}
	n++
	if c, err = w.Write(body); err != nil {
		return
	}
	n += c
	return
}

// Encode encodes routing prefix and message
func Encode(w io.Writer, addrs []uint8, msgID uint32, bodyFlag uint8, body []byte) (n int, err error) {
	var c int
	if c, err = EncodeRoute(w, addrs); err != nil {
		return
	}
	n += c
	if c, err = EncodeMsg(w, msgID, bodyFlag, body); err != nil {
		return
	}
	n += c
	return
}

// EncodeAsMsg encodes into new Msg
func EncodeAsMsg(addrs []uint8, msgID uint32, bodyFlag uint8, body []byte) (msg *Msg, err error) {
	var buf bytes.Buffer
	msg = &Msg{}
	msg.Head.MsgID = msgID
	msg.Head.Flag = Format
	msg.Head.BodyBytes = uint16(len(body) + 1)
	msg.Body.Flag = bodyFlag
	if addrs != nil {
		if _, err = EncodeRoute(&buf, addrs); err != nil {
			return
		}
	}
	if _, err = EncodeMsg(&buf, msgID, bodyFlag, body); err != nil {
		return
	}
	msg.Head.Raw = buf.Bytes()
	bodyOff := len(msg.Head.Raw) - int(msg.Head.BodyBytes)
	if addrs != nil {
		msg.Head.Prefix = msg.Head.Raw[0]
		msg.Head.Addrs = msg.Head.Raw[1 : len(addrs)+1]
		msg.Head.RawPrefix = msg.Head.Raw[0 : len(addrs)+1]
		msg.Head.RawHead = msg.Head.Raw[len(addrs)+1 : bodyOff]
	} else {
		msg.Head.RawHead = msg.Head.Raw[0:bodyOff]
	}
	msg.Body.Raw = msg.Head.Raw[bodyOff:]
	msg.Body.Data = msg.Body.Raw[1:]
	return
}

func encode7bit(w io.Writer, val uint64) (n int, err error) {
	data := make([]byte, 1)
	for {
		data[0] = uint8(val & 0x7f)
		val >>= 7
		if val != 0 {
			data[0] |= 0x80
		}
		if _, err = w.Write(data); err != nil {
			return
		}
		n++
		if val == 0 {
			break
		}
	}
	return
}

func decode7bit(w io.Writer, r io.Reader, maxBytes int) (val uint64, err error) {
	data := make([]byte, 1)
	for i := 0; i < maxBytes; i++ {
		if _, err = io.ReadFull(r, data); err != nil {
			return
		}
		if _, err = w.Write(data); err != nil {
			return
		}
		val |= (uint64(data[0]) & 0x7f) << (7 * uint(i))
		if (data[0] & 0x80) == 0 {
			break
		}
	}
	return
}

func decode7bit32(w io.Writer, r io.Reader) (uint32, error) {
	val, err := decode7bit(w, r, 4)
	return uint32(val), err
}

func decode7bit16(w io.Writer, r io.Reader) (uint16, error) {
	val, err := decode7bit(w, r, 2)
	return uint16(val), err
}
