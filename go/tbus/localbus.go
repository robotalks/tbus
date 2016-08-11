package tbus

import (
	"fmt"
	"sync"

	prot "github.com/evo-bots/tbus/go/tbus/protocol"
	bitset "github.com/willf/bitset"
)

// LocalBus implements a bus manages local devices
type LocalBus struct {
	device  *BusDev
	host    *localBusHost
	slave   *localBusSlave
	addrs   *bitset.BitSet
	devices map[uint8]Device
	lock    sync.RWMutex
}

type localBusHost struct {
	bus   *LocalBus
	msgCh chan prot.Msg
}

type localBusSlave struct {
	bus *LocalBus
}

// NewLocalBus creates a local bus
func NewLocalBus() *LocalBus {
	b := &LocalBus{
		addrs:   BitsBucket(),
		devices: make(map[uint8]Device),
	}
	b.device = NewBusDev(b)
	b.host = &localBusHost{bus: b, msgCh: make(chan prot.Msg, 1)}
	b.slave = &localBusSlave{bus: b}
	b.addrs.SetTo(0, false)
	b.device.Attach(b.slave, 0)
	return b
}

// Device returns the bus device
func (b *LocalBus) Device() *BusDev {
	return b.device
}

// Host implements Bus
func (b *LocalBus) Host() BusHost {
	return b.host
}

// Slave implements Bus
func (b *LocalBus) Slave() BusSlave {
	return b.slave
}

// Plug implements Bus
func (b *LocalBus) Plug(dev Device) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	index, found := b.addrs.NextSet(0)
	if found {
		addr := uint8(index)
		b.addrs.SetTo(index, false)
		b.devices[addr] = dev
		err := dev.Attach(b.slave, addr)
		if err != nil {
			delete(b.devices, addr)
			b.addrs.SetTo(index, true)
			return err
		}
	} else {
		return ErrAddrNotAvail
	}
	return nil
}

// Unplug implements Bus
func (b *LocalBus) Unplug(dev Device) error {
	addr := dev.Address()
	if addr != 0 {
		b.lock.Lock()
		defer b.lock.Unlock()
		err := dev.Detach()
		if err != nil {
			return err
		}
		delete(b.devices, addr)
		b.addrs.SetTo(uint(addr), true)
	}
	return nil
}

// RouteMsg implements BusLogic
func (b *LocalBus) RouteMsg(msg *prot.Msg) error {
	addr := msg.Head.Addrs[0]
	b.lock.RLock()
	device := b.devices[addr]
	b.lock.RUnlock()
	if device == nil {
		return ErrInvalidAddr
	}
	msg.Head.Addrs = msg.Head.Addrs[1:]
	if len(msg.Head.Addrs) == 0 {
		msg.Head.Prefix = 0
		msg.Head.Raw = msg.Head.Raw[2:]
		msg.Head.PrefixRaw = msg.Head.Raw[0:0]
	} else {
		msg.Head.Raw = msg.Head.Raw[1:]
		msg.Head.Raw[0] = msg.Head.Prefix
		msg.Head.PrefixRaw = msg.Head.Raw[0 : len(msg.Head.Addrs)+1]
	}
	return device.SendMsg(msg)
}

// Enumerate implements BusLogic
func (b *LocalBus) Enumerate() (*BusEnumeration, error) {
	enum := &BusEnumeration{}
	b.lock.RLock()
	defer b.lock.RUnlock()
	for _, dev := range b.devices {
		enum.Devices = append(enum.Devices, &DeviceInfo{
			Address:  uint32(dev.Address()),
			ClassId:  dev.ClassID(),
			DeviceId: dev.DeviceID(),
		})
	}
	return enum, nil
}

// Forward implements BusLogic
func (b *LocalBus) Forward(*ForwardMsg) error {
	return fmt.Errorf("not implemented")
}

func (h *localBusHost) MsgChan() <-chan prot.Msg {
	return h.msgCh
}

func (h *localBusHost) SendMsg(msg *prot.Msg) error {
	return h.bus.device.SendMsg(msg)
}

func (s *localBusSlave) SendMsg(msg *prot.Msg) error {
	s.bus.host.msgCh <- *msg
	return nil
}
