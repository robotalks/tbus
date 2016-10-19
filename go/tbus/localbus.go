package tbus

import (
	"sort"
	"sync"

	bitset "github.com/willf/bitset"
)

// LocalBus implements BusLogic and manages local devices
type LocalBus struct {
	LogicBase
	port    localBusPort
	addrs   *bitset.BitSet
	devices map[uint8]Device
	lock    sync.RWMutex
}

type localBusPort struct {
	bus *LocalBus
}

// NewLocalBus creates a local bus
func NewLocalBus() *LocalBus {
	b := &LocalBus{
		addrs:   BitsBucket(),
		devices: make(map[uint8]Device),
	}
	b.port.bus = b
	b.addrs.SetTo(0, false)
	return b
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
		dev.AttachTo(&b.port, addr)
	} else {
		return ErrAddrNotAvail
	}
	return nil
}

// Unplug implements Bus
func (b *LocalBus) Unplug(dev Device) error {
	addr := uint8(dev.DeviceInfo().Address)
	if addr != 0 {
		b.lock.Lock()
		defer b.lock.Unlock()
		dev.AttachTo(nil, 0)
		delete(b.devices, addr)
		b.addrs.SetTo(uint(addr), true)
	}
	return nil
}

// RouteMsg implements BusLogic
func (b *LocalBus) RouteMsg(msg *Msg) error {
	addr := msg.Head.Addrs[0]
	b.lock.RLock()
	device := b.devices[addr]
	b.lock.RUnlock()
	if device == nil {
		return SendReply(b.Device.BusPort(), msg.Head.MsgID, nil, ErrInvalidAddr)
	}
	msg.Head.Addrs = msg.Head.Addrs[1:]
	return device.DispatchMsg(msg)
}

// DeviceInfoListByAddr is the alias of []*DeviceInfo
// with sorting implementation by address
type DeviceInfoListByAddr []*DeviceInfo

func (l DeviceInfoListByAddr) Len() int           { return len(l) }
func (l DeviceInfoListByAddr) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l DeviceInfoListByAddr) Less(i, j int) bool { return l[i].Address < l[j].Address }

// Enumerate implements BusLogic
func (b *LocalBus) Enumerate() (*BusEnumeration, error) {
	enum := &BusEnumeration{}
	b.lock.RLock()
	for _, dev := range b.devices {
		info := dev.DeviceInfo()
		enum.Devices = append(enum.Devices, &info)
	}
	b.lock.RUnlock()
	sort.Sort(DeviceInfoListByAddr(enum.Devices))
	return enum, nil
}

func (b *LocalBus) sendToHost(msg *Msg) error {
	if b.Device == nil {
		return ErrNoAssocDevice
	}
	return b.Device.BusPort().DispatchMsg(msg)
}

func (s *localBusPort) DispatchMsg(msg *Msg) error {
	return s.bus.sendToHost(msg)
}
