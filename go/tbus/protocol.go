package tbus

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
)

// Prefix and Flag
const (
	PfxRoutingMask    uint8 = 0xe0
	PfxRouting        uint8 = 0xc0
	PfxRoutingAddrNum uint8 = 0x1f

	RoutingAddrsMax = 32

	FormatMask uint8 = 0xf0
	Format     uint8 = 0x10 // rev 1, protobuf encoded
	EventMask  uint8 = 0x01

	BodyError uint8 = 0x80 // body contains error
)

// RouteAddr is routable address
type RouteAddr []uint8

// RouteWith build RouteAddr from multiple addresses
func RouteWith(addrs ...uint8) RouteAddr {
	return RouteAddr(addrs)
}

// Prefix prefixes address to current address
func (a RouteAddr) Prefix(addrs ...uint8) RouteAddr {
	return append(addrs, a...)
}

// MsgID is message ID
type MsgID []byte

// MsgIDVarInt generates variant len msg ID from int
func MsgIDVarInt(val uint32) MsgID {
	var buf bytes.Buffer
	encode7bit(&buf, val)
	return MsgID(buf.Bytes())
}

// Extend extends MsgID with more bytes
func (id MsgID) Extend(a []byte) MsgID {
	return append(a, id...)
}

// Extract extracts extra bytes from existing MsgID
func (id MsgID) Extract(bytes uint32) (data []byte, newID MsgID) {
	if bytes >= uint32(len(id)) {
		return id, nil
	}
	return id[:bytes], id[bytes:]
}

// VarInt decode the msg ID as variant len int
func (id MsgID) VarInt() (uint32, error) {
	return decode7bit32(bytes.NewBuffer(id))
}

// MsgHead is message head
type MsgHead struct {
	Prefix    uint8
	Addrs     RouteAddr
	Flag      uint8
	BodyBytes uint32
	MsgID     MsgID
}

// NeedRoute indicates the message contains routing addresses
func (h *MsgHead) NeedRoute() bool {
	return len(h.Addrs) > 0
}

// IsEvent indicates this is an event from device to master
func (h *MsgHead) IsEvent() bool {
	return (h.Flag & EventMask) != 0
}

// EncodeTo encodes header to a writer
func (h *MsgHead) EncodeTo(w io.Writer) (err error) {
	buf := bufio.NewWriter(w)
	h.Prefix &^= PfxRouting | PfxRoutingAddrNum
	if l := len(h.Addrs); l > 0 {
		if l > RoutingAddrsMax {
			panic("too many addrs")
		}
		h.Prefix |= PfxRouting | uint8(l-1)
		if err = buf.WriteByte(h.Prefix); err == nil {
			_, err = buf.Write(h.Addrs)
		}
	}
	if err == nil {
		err = buf.WriteByte(h.Flag)
	}
	if err == nil {
		_, err = encode7bit(buf, uint32(len(h.MsgID)))
	}
	if err == nil {
		_, err = encode7bit(buf, h.BodyBytes)
	}
	if err == nil && len(h.MsgID) > 0 {
		_, err = buf.Write(h.MsgID)
	}
	if err == nil {
		err = buf.Flush()
	}
	return
}

// MsgBody represents encoded msg body
type MsgBody struct {
	Flag uint8
	Data []byte
}

// IsError indicates the body is an error reply
func (b *MsgBody) IsError() bool {
	return (b.Flag & BodyError) != 0
}

// Decode decodes the body as message or error
func (b *MsgBody) Decode(val proto.Message) error {
	if b.IsError() {
		replyErr := &Error{}
		if err := proto.Unmarshal(b.Data, replyErr); err != nil {
			return err
		}
		return replyErr
	}
	if val != nil {
		return proto.Unmarshal(b.Data, val)
	}
	return nil
}

// Msg is a completely message
type Msg struct {
	Head MsgHead
	Body MsgBody
}

// EncodeTo encodes the whole message to a writer
func (m *Msg) EncodeTo(w io.Writer) error {
	buf := bufio.NewWriter(w)
	m.Head.BodyBytes = uint32(len(m.Body.Data) + 1)
	err := m.Head.EncodeTo(buf)
	if err == nil {
		err = buf.WriteByte(m.Body.Flag)
		if err == nil {
			_, err = buf.Write(m.Body.Data)
		}
	}
	if err == nil {
		err = buf.Flush()
	}
	return err
}

// Dispatch dispatches the message to a dispatcher
func (m *Msg) Dispatch(dispatcher MsgDispatcher) error {
	return dispatcher.DispatchMsg(m)
}

// DecodeHead decodes message head from input
func DecodeHead(reader io.Reader) (head MsgHead, err error) {
	buf := make([]byte, 1)
	if _, err = io.ReadAtLeast(reader, buf, 1); err != nil {
		return
	}
	head.Prefix = buf[0]

	var addrNum int
	if (head.Prefix & PfxRoutingMask) == PfxRouting {
		addrNum = int((head.Prefix & PfxRoutingAddrNum) + 1)
		head.Addrs = make([]byte, addrNum)
		if _, err = io.ReadAtLeast(reader, head.Addrs, addrNum); err != nil {
			return
		}
		if _, err = io.ReadAtLeast(reader, buf, 1); err != nil {
			return
		}
		head.Flag = buf[0]
	} else {
		head.Flag = head.Prefix
		head.Prefix = 0
	}

	if (head.Flag & FormatMask) != Format {
		err = fmt.Errorf("unknown byte 0x%02x", head.Flag)
		return
	}

	var msgIDBytes uint16
	if msgIDBytes, err = decode7bit16(reader); err != nil {
		return
	}
	if head.BodyBytes, err = decode7bit32(reader); err != nil {
		return
	}

	if msgIDBytes > 0 {
		head.MsgID = make([]byte, msgIDBytes)
		if _, err = io.ReadAtLeast(reader, head.MsgID, int(msgIDBytes)); err != nil {
			return
		}
	}
	return
}

// DecodeBody decodes message body from input
func DecodeBody(reader io.Reader, bodyBytes uint32) (body MsgBody, err error) {
	if bodyBytes == 0 {
		return
	}
	raw := make([]byte, bodyBytes)
	if _, err = io.ReadAtLeast(reader, raw, int(bodyBytes)); err != nil {
		return
	}
	body.Flag = uint8(raw[0])
	body.Data = raw[1:]
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

// DecodeAs decodes a complete message and unmarshals the protobuf message
func DecodeAs(reader io.Reader, val proto.Message) (msg Msg, err error) {
	if msg, err = Decode(reader); err != nil {
		return
	}
	err = msg.Body.Decode(val)
	return msg, err
}

// MsgBuilder is a helper to build a message
type MsgBuilder struct {
	msg *Msg
}

// BuildMsg creates a MsgBuilder
func BuildMsg() *MsgBuilder {
	return &MsgBuilder{msg: &Msg{Head: MsgHead{Flag: Format}}}
}

// RouteTo specifies the routing addresses
func (b *MsgBuilder) RouteTo(addrs RouteAddr) *MsgBuilder {
	b.msg.Head.Addrs = addrs
	return b
}

// MsgID specifies the message ID
func (b *MsgBuilder) MsgID(msgID MsgID) *MsgBuilder {
	b.msg.Head.MsgID = msgID
	return b
}

// MsgIDVarInt specifies the message ID using variant len int
func (b *MsgBuilder) MsgIDVarInt(val uint32) *MsgBuilder {
	return b.MsgID(MsgIDVarInt(val))
}

// Body specifies the body
func (b *MsgBuilder) Body(flag uint8, data []byte) *MsgBuilder {
	b.msg.Body.Flag = flag
	b.msg.Body.Data = data
	return b
}

// EncodeBody specifies the body by encoding a protobuf message
func (b *MsgBuilder) EncodeBody(flag uint8, val proto.Message) *MsgBuilder {
	if val == nil {
		val = &empty.Empty{}
	}
	encoded, err := proto.Marshal(val)
	if err != nil {
		panic(err)
	}
	return b.Body(flag, encoded)
}

// EncodeEvent specifies this is an event msg and encode the event into body
func (b *MsgBuilder) EncodeEvent(addr, channelID uint8, val proto.Message) *MsgBuilder {
	b.msg.Head.Addrs = []uint8{addr}
	b.msg.Head.Flag |= EventMask
	b.EncodeBody(channelID, val)
	return b
}

// Build finalizes the message
func (b *MsgBuilder) Build() (msg *Msg) {
	msg = b.msg
	b.msg = nil
	return
}

func encode7bit(w io.Writer, val uint32) (n int, err error) {
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

func decode7bit(r io.Reader, maxBytes int) (val uint64, err error) {
	data := make([]byte, 1)
	for i := 0; i < maxBytes; i++ {
		if _, err = io.ReadFull(r, data); err != nil {
			return
		}
		val |= (uint64(data[0]) & 0x7f) << (7 * uint(i))
		if (data[0] & 0x80) == 0 {
			break
		}
	}
	if (data[0] & 0x80) != 0 {
		err = fmt.Errorf("size to large")
	}
	return
}

func decode7bit32(r io.Reader) (uint32, error) {
	val, err := decode7bit(r, 4)
	return uint32(val), err
}

func decode7bit16(r io.Reader) (uint16, error) {
	val, err := decode7bit(r, 2)
	return uint16(val), err
}
