package tbus

import (
	prot "github.com/evo-bots/tbus/go/tbus/protocol"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
)

// DeviceBase implements basic device operations
type DeviceBase struct {
	Info DeviceInfo

	busPort BusPort
}

// Address returns current address
func (d *DeviceBase) Address() uint8 {
	return uint8(d.Info.Address)
}

// ClassID implements Device
func (d *DeviceBase) ClassID() uint32 {
	return d.Info.ClassId
}

// DeviceID implements Device
func (d *DeviceBase) DeviceID() uint32 {
	return d.Info.DeviceId
}

// AttachTo implements Device
func (d *DeviceBase) AttachTo(busPort BusPort, addr uint8) {
	d.busPort = busPort
	d.Info.Address = uint32(addr)
}

// BusPort implements Device
func (d *DeviceBase) BusPort() BusPort {
	return d.busPort
}

// Reply write reply to bus
func (d *DeviceBase) Reply(msgID uint32, reply proto.Message) error {
	if reply == nil {
		reply = &empty.Empty{}
	}

	encoded, err := proto.Marshal(reply)
	if err != nil {
		return err
	}
	msg, err := prot.EncodeAsMsg(nil, msgID, 0, encoded)
	if err != nil {
		return err
	}
	return d.busPort.SendMsg(msg)
}

// LogicBase implements DeviceLogic
type LogicBase struct {
	Device Device
}

// SetDevice implements DeviceLogic
func (l *LogicBase) SetDevice(dev Device) {
	l.Device = dev
}
