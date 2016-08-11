package tbus

import (
	prot "github.com/evo-bots/tbus/go/tbus/protocol"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
)

// DeviceBase implements basic device operations
type DeviceBase struct {
	Info DeviceInfo
	Bus  BusSlave
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

// Attach implements Device
func (d *DeviceBase) Attach(bus BusSlave, addr uint8) error {
	if d.Bus != nil {
		panic("already attached to bus")
	}
	d.Bus = bus
	d.Info.Address = uint32(addr)
	return nil
}

// Detach implements Device
func (d *DeviceBase) Detach() error {
	if d.Bus != nil {
		d.Bus = nil
		d.Info.Address = 0
	}
	return nil
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
	return d.Bus.SendMsg(msg)
}
