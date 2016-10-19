package tbus

import (
	proto "github.com/golang/protobuf/proto"
)

// DeviceBase implements basic device operations
type DeviceBase struct {
	Info DeviceInfo

	busPort BusPort
}

// DeviceInfo returns device information
func (d *DeviceBase) DeviceInfo() DeviceInfo {
	return d.Info
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
func (d *DeviceBase) Reply(msgID MsgID, reply proto.Message, err error) error {
	return SendReply(d.busPort, msgID, reply, err)
}

// SendReply sends back reply
func SendReply(dispatcher MsgDispatcher, msgID MsgID, reply proto.Message, err error) error {
	if dispatcher == nil {
		return ErrInvalidDispatcher
	}
	flag := uint8(0)
	if err != nil {
		flag |= BodyError
		reply = &Error{Message: err.Error()}
	}

	return BuildMsg().
		MsgID(msgID).
		EncodeBody(flag, reply).
		Build().
		Dispatch(dispatcher)
}

// LogicBase implements DeviceLogic
type LogicBase struct {
	Device Device
}

// SetDevice implements DeviceLogic
func (l *LogicBase) SetDevice(dev Device) {
	l.Device = dev
}

// DeviceAddress is a helper to construct device address
func DeviceAddress(devs ...Device) []uint8 {
	addrs := make([]uint8, len(devs))
	for n, dev := range devs {
		addrs[n] = uint8(dev.DeviceInfo().Address)
	}
	return addrs
}

// DeviceAddress returns a full device address
func (d *DeviceInfo) DeviceAddress() []uint8 {
	return []uint8{uint8(d.Address)}
}

// AddLabel adds a single label to device info
func (d *DeviceInfo) AddLabel(name, value string) *DeviceInfo {
	m := d.Labels
	if m == nil {
		m = make(map[string]string)
		d.Labels = m
	}
	m[name] = value
	return d
}

// AddLabels adds multiple labels
func (d *DeviceInfo) AddLabels(labels map[string]string) *DeviceInfo {
	if d.Labels == nil {
		d.Labels = labels
	} else {
		for k, v := range labels {
			d.Labels[k] = v
		}
	}
	return d
}
