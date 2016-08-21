// Code generated by protoc-gen-go.
// source: tbus/motor.proto
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

type MotorDriveState_Direction int32

const (
	MotorDriveState_Fwd MotorDriveState_Direction = 0
	MotorDriveState_Rev MotorDriveState_Direction = 1
)

var MotorDriveState_Direction_name = map[int32]string{
	0: "Fwd",
	1: "Rev",
}
var MotorDriveState_Direction_value = map[string]int32{
	"Fwd": 0,
	"Rev": 1,
}

func (x MotorDriveState_Direction) String() string {
	return proto.EnumName(MotorDriveState_Direction_name, int32(x))
}
func (MotorDriveState_Direction) EnumDescriptor() ([]byte, []int) { return fileDescriptor3, []int{0, 0} }

type MotorDriveState struct {
	Direction MotorDriveState_Direction `protobuf:"varint,1,opt,name=direction,enum=tbus.MotorDriveState_Direction" json:"direction,omitempty"`
	Speed     uint32                    `protobuf:"varint,2,opt,name=speed" json:"speed,omitempty"`
}

func (m *MotorDriveState) Reset()                    { *m = MotorDriveState{} }
func (m *MotorDriveState) String() string            { return proto.CompactTextString(m) }
func (*MotorDriveState) ProtoMessage()               {}
func (*MotorDriveState) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{0} }

type MotorBrakeState struct {
	On bool `protobuf:"varint,1,opt,name=on" json:"on,omitempty"`
}

func (m *MotorBrakeState) Reset()                    { *m = MotorBrakeState{} }
func (m *MotorBrakeState) String() string            { return proto.CompactTextString(m) }
func (*MotorBrakeState) ProtoMessage()               {}
func (*MotorBrakeState) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{1} }

func init() {
	proto.RegisterType((*MotorDriveState)(nil), "tbus.MotorDriveState")
	proto.RegisterType((*MotorBrakeState)(nil), "tbus.MotorBrakeState")
	proto.RegisterEnum("tbus.MotorDriveState_Direction", MotorDriveState_Direction_name, MotorDriveState_Direction_value)
}

func init() { proto.RegisterFile("tbus/motor.proto", fileDescriptor3) }

var fileDescriptor3 = []byte{
	// 273 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x7c, 0x90, 0xc1, 0x4e, 0x83, 0x40,
	0x10, 0x86, 0x5d, 0x0a, 0x6a, 0x27, 0xb1, 0x92, 0x8d, 0x1a, 0xc4, 0x18, 0x91, 0x53, 0x4f, 0x4b,
	0x52, 0xaf, 0xf5, 0x62, 0xaa, 0x37, 0x2f, 0xf0, 0x04, 0x50, 0xd6, 0x86, 0x28, 0xcc, 0x66, 0x99,
	0xd6, 0x78, 0xf3, 0xe2, 0x23, 0xf9, 0x0c, 0xbe, 0x96, 0xd9, 0x85, 0xda, 0x68, 0x6c, 0x2f, 0x84,
	0x9d, 0xf9, 0xfe, 0xf9, 0xff, 0xfc, 0xe0, 0x53, 0xb1, 0x6c, 0x93, 0x1a, 0x09, 0xb5, 0x50, 0x1a,
	0x09, 0xb9, 0x6b, 0x26, 0xe1, 0xc5, 0x02, 0x71, 0xf1, 0x22, 0x13, 0x3b, 0x2b, 0x96, 0x4f, 0x89,
	0xac, 0x15, 0xbd, 0x75, 0x48, 0x78, 0x6e, 0x45, 0x73, 0xac, 0x6b, 0x6c, 0x12, 0x54, 0x54, 0x61,
	0xd3, 0x76, 0xab, 0xf8, 0x83, 0xc1, 0xf1, 0xa3, 0xb9, 0x36, 0xd3, 0xd5, 0x4a, 0x66, 0x94, 0x93,
	0xe4, 0xb7, 0x30, 0x2c, 0x2b, 0x2d, 0xe7, 0x86, 0x0b, 0x58, 0xc4, 0xc6, 0xa3, 0xc9, 0x95, 0x30,
	0x27, 0xc4, 0x1f, 0x52, 0xcc, 0xd6, 0x58, 0xba, 0x51, 0xf0, 0x13, 0xf0, 0x5a, 0x25, 0x65, 0x19,
	0x38, 0x11, 0x1b, 0x1f, 0xa5, 0xdd, 0x23, 0xbe, 0x84, 0xe1, 0x0f, 0xcd, 0x0f, 0x60, 0xf0, 0xf0,
	0x5a, 0xfa, 0x7b, 0xe6, 0x27, 0x95, 0x2b, 0x9f, 0xc5, 0xd7, 0x7d, 0x8c, 0x3b, 0x9d, 0x3f, 0xf7,
	0x31, 0x46, 0xe0, 0xf4, 0xfe, 0x87, 0xa9, 0x83, 0xcd, 0xe4, 0x8b, 0x81, 0x67, 0x19, 0x3e, 0x05,
	0x2f, 0xa3, 0x5c, 0x13, 0x3f, 0xfd, 0x37, 0x56, 0x78, 0x26, 0xba, 0x36, 0xc4, 0xba, 0x0d, 0x71,
	0x6f, 0xda, 0x88, 0xdd, 0xf7, 0xcf, 0x80, 0xf1, 0x29, 0xb8, 0x19, 0xa1, 0xe2, 0x5b, 0xa8, 0x9d,
	0x6a, 0xc7, 0x78, 0xdb, 0x8c, 0xbf, 0xbc, 0x37, 0xa9, 0x77, 0xaa, 0x07, 0xa1, 0xf9, 0x46, 0xc5,
	0xbe, 0xdd, 0xdd, 0x7c, 0x07, 0x00, 0x00, 0xff, 0xff, 0xbc, 0xa5, 0x88, 0x9a, 0xcd, 0x01, 0x00,
	0x00,
}

//
// GENERTED FROM tbus/motor.proto, DO NOT EDIT
//

// MotorClassID is the class ID of Motor
const MotorClassID uint32 = 0x0020

// MotorLogic defines the logic interface
type MotorLogic interface {
	DeviceLogic
    Start(*MotorDriveState) error
    Stop() error
    Brake(*MotorBrakeState) error
}

// MotorDev is the device
type MotorDev struct {
    DeviceBase
    Logic MotorLogic
}

// NewMotorDev creates a new device
func NewMotorDev(logic MotorLogic) *MotorDev {
    d := &MotorDev{Logic: logic}
    d.Info.ClassId = MotorClassID
	logic.SetDevice(d)
    return d
}

// SendMsg implements Device
func (d *MotorDev) SendMsg(msg *prot.Msg) (err error) {
	if msg.Head.NeedRoute() {
		return d.Reply(msg.Head.MsgID, nil, ErrRouteNotSupport)
	}
    var reply proto.Message
	switch msg.Body.Flag {
    case 1: // Start
        params := &MotorDriveState{}
        err = proto.Unmarshal(msg.Body.Data, params)
        if err == nil {
            err = d.Logic.Start(params)
        }
    case 2: // Stop
        err = d.Logic.Stop()
    case 3: // Brake
        params := &MotorBrakeState{}
        err = proto.Unmarshal(msg.Body.Data, params)
        if err == nil {
            err = d.Logic.Brake(params)
        }
    default:
        err = ErrInvalidMethod
    }
    return d.Reply(msg.Head.MsgID, reply, err)
}

// SetDeviceID sets device id
func (d *MotorDev) SetDeviceID(id uint32) *MotorDev {
	d.Info.DeviceId = id
	return d
}

// MotorCtl is the device controller
type MotorCtl struct {
    Controller
}

// NewMotorCtl creates controller for Motor
func NewMotorCtl(master Master) *MotorCtl {
	c := &MotorCtl{}
	c.Master = master
	return c
}

// SetAddress sets routing address for target device
func (c *MotorCtl) SetAddress(addrs []uint8) *MotorCtl {
	c.Address = addrs
	return c
}

// Start wraps class Motor
func (c *MotorCtl) Start(params *MotorDriveState) error {
	return c.Invoke(1, params, nil)
}

// Stop wraps class Motor
func (c *MotorCtl) Stop() error {
	return c.Invoke(2, nil, nil)
}

// Brake wraps class Motor
func (c *MotorCtl) Brake(params *MotorBrakeState) error {
	return c.Invoke(3, params, nil)
}

