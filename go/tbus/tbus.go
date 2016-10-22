package tbus

import (
	"fmt"
	"io"
	"time"

	proto "github.com/golang/protobuf/proto"
)

var (
	// ErrInvalidMethod indicates method index is invalid
	ErrInvalidMethod = fmt.Errorf("invalid method index")
	// ErrInvalidAddr indicates address doesn't map to a device
	ErrInvalidAddr = fmt.Errorf("invalid address")
	// ErrRouteNotSupport indicates the device doesn't support routing
	ErrRouteNotSupport = fmt.Errorf("route not supported")
	// ErrRecvAborted indicates the receiving is cancelled
	ErrRecvAborted = fmt.Errorf("receiving aborted")
	// ErrRecvTimeout indicates the receiving is timed out
	ErrRecvTimeout = fmt.Errorf("receiving timed out")
	// ErrRecvEnd indicates the receiving is ended
	ErrRecvEnd = io.EOF
	// ErrAddrNotAvail indicates no more address can be allocated
	ErrAddrNotAvail = fmt.Errorf("address not available")
	// ErrNoAssocDevice indicates a logic is not associated with device
	ErrNoAssocDevice = fmt.Errorf("logic not associated with device")
	// ErrInvalidDispatcher indicates dispatcher is unavailable
	ErrInvalidDispatcher = fmt.Errorf("dispatcher not available")
)

// MsgReceiver provides a message chan for read
type MsgReceiver interface {
	MsgChan() <-chan Msg
}

// MsgDispatcher dispatches a message to target
type MsgDispatcher interface {
	DispatchMsg(*Msg) error
}

// MsgRouter is able to route a message
type MsgRouter interface {
	RouteMsg(*Msg) error
}

// BusPort is the device side of the bus
type BusPort interface {
	MsgDispatcher
}

// Bus defines a bus instance
type Bus interface {
	Plug(Device) error
	Unplug(Device) error
}

// Device defines a device instance
type Device interface {
	MsgDispatcher
	DeviceInfo() DeviceInfo
	AttachTo(BusPort, uint8)
	BusPort() BusPort
}

// DeviceLogic implements device functions
type DeviceLogic interface {
	SetDevice(Device)
}

// Master is the bus master
type Master interface {
	Invoke(method uint8, params proto.Message, addrs RouteAddr) Invocation
	Subscribe(channel uint8, addrs RouteAddr, handler EventHandler) EventSubscription
}

// Invocation represents the result of method invocation
type Invocation interface {
	Recv() (MsgReceiver, error)
	MsgID() MsgID
	Timeout(time.Duration) Invocation
	Result(proto.Message) error
	Ignore()
}

// EventSubscription is the subscription to an event channel
type EventSubscription interface {
	io.Closer
}

// Event is an received event
type Event interface {
	Channel() uint8
	Address() RouteAddr
	Decode(proto.Message) error
}

// EventHandler handles subscribed events
type EventHandler interface {
	HandleEvent(Event, EventSubscription)
}
