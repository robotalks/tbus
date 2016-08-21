// Code generated by protoc-gen-go.
// source: tbus/servo.proto
// DO NOT EDIT!

package tbus

import prot "github.com/robotalks/tbus/go/tbus/protocol"
import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/golang/protobuf/ptypes/empty"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type ServoPosition struct {
	Angle uint32 `protobuf:"varint,1,opt,name=angle" json:"angle,omitempty"`
}

func (m *ServoPosition) Reset()                    { *m = ServoPosition{} }
func (m *ServoPosition) String() string            { return proto.CompactTextString(m) }
func (*ServoPosition) ProtoMessage()               {}
func (*ServoPosition) Descriptor() ([]byte, []int) { return fileDescriptor4, []int{0} }

func init() {
	proto.RegisterType((*ServoPosition)(nil), "tbus.ServoPosition")
}

func init() { proto.RegisterFile("tbus/servo.proto", fileDescriptor4) }

var fileDescriptor4 = []byte{
	// 184 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0x12, 0x28, 0x49, 0x2a, 0x2d,
	0xd6, 0x2f, 0x4e, 0x2d, 0x2a, 0xcb, 0xd7, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x01, 0x89,
	0x48, 0x49, 0xa7, 0xe7, 0xe7, 0xa7, 0xe7, 0xa4, 0xea, 0x83, 0xc5, 0x92, 0x4a, 0xd3, 0xf4, 0x53,
	0x73, 0x0b, 0x4a, 0x2a, 0x21, 0x4a, 0xa4, 0x24, 0xc1, 0x9a, 0x92, 0xf3, 0x73, 0x73, 0xf3, 0xf3,
	0xf4, 0xf3, 0x0b, 0x4a, 0x32, 0xf3, 0xf3, 0x8a, 0x21, 0x52, 0x4a, 0xaa, 0x5c, 0xbc, 0xc1, 0x20,
	0xc3, 0x02, 0xf2, 0x8b, 0x33, 0x41, 0xe2, 0x42, 0x22, 0x5c, 0xac, 0x89, 0x79, 0xe9, 0x39, 0xa9,
	0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0xbc, 0x41, 0x10, 0x8e, 0x51, 0x2f, 0x23, 0x17, 0x2b, 0x58, 0x9d,
	0x90, 0x03, 0x17, 0x77, 0x70, 0x6a, 0x09, 0x5c, 0xb9, 0xb0, 0x1e, 0xc8, 0x6c, 0x3d, 0x14, 0x33,
	0xa4, 0xc4, 0xf4, 0x20, 0xae, 0xd1, 0x83, 0xb9, 0x46, 0xcf, 0x15, 0xe4, 0x1a, 0x25, 0x96, 0x86,
	0xad, 0x12, 0x8c, 0x42, 0x36, 0x5c, 0x2c, 0xc1, 0x25, 0xf9, 0x05, 0x42, 0x38, 0x54, 0xe1, 0xd5,
	0xcd, 0x24, 0x05, 0x22, 0x55, 0x92, 0xd8, 0xc0, 0x72, 0xc6, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff,
	0x76, 0x2c, 0x2c, 0xa2, 0x0f, 0x01, 0x00, 0x00,
}

//
// GENERTED FROM tbus/servo.proto, DO NOT EDIT
//

// ServoClassID is the class ID of Servo
const ServoClassID uint32 = 0x0024

// ServoLogic defines the logic interface
type ServoLogic interface {
	DeviceLogic
    SetPosition(*ServoPosition) error
    Stop() error
}

// ServoDev is the device
type ServoDev struct {
    DeviceBase
    Logic ServoLogic
}

// NewServoDev creates a new device
func NewServoDev(logic ServoLogic) *ServoDev {
    d := &ServoDev{Logic: logic}
    d.Info.ClassId = ServoClassID
	logic.SetDevice(d)
    return d
}

// SendMsg implements Device
func (d *ServoDev) SendMsg(msg *prot.Msg) (err error) {
	if msg.Head.NeedRoute() {
		return d.Reply(msg.Head.MsgID, nil, ErrRouteNotSupport)
	}
    var reply proto.Message
	switch msg.Body.Flag {
    case 1: // SetPosition
        params := &ServoPosition{}
        err = proto.Unmarshal(msg.Body.Data, params)
        if err == nil {
            err = d.Logic.SetPosition(params)
        }
    case 2: // Stop
        err = d.Logic.Stop()
    default:
        err = ErrInvalidMethod
    }
    return d.Reply(msg.Head.MsgID, reply, err)
}

// SetDeviceID sets device id
func (d *ServoDev) SetDeviceID(id uint32) *ServoDev {
	d.Info.DeviceId = id
	return d
}

// ServoCtl is the device controller
type ServoCtl struct {
    Controller
}

// NewServoCtl creates controller for Servo
func NewServoCtl(master Master) *ServoCtl {
	c := &ServoCtl{}
	c.Master = master
	return c
}

// SetAddress sets routing address for target device
func (c *ServoCtl) SetAddress(addrs []uint8) *ServoCtl {
	c.Address = addrs
	return c
}

// SetPosition wraps class Servo
func (c *ServoCtl) SetPosition(params *ServoPosition) error {
	return c.Invoke(1, params, nil)
}

// Stop wraps class Servo
func (c *ServoCtl) Stop() error {
	return c.Invoke(2, nil, nil)
}

