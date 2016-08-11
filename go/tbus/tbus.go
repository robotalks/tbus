package tbus

import (
	"fmt"
	"io"
	"time"

	prot "github.com/evo-bots/tbus/go/tbus/protocol"
	proto "github.com/golang/protobuf/proto"
)

var (
	// ErrInvalidMethod indicates method index is invalid
	ErrInvalidMethod = fmt.Errorf("invalid method index")
	// ErrInvalidAddr indicates address doesn't map to a device
	ErrInvalidAddr = fmt.Errorf("invalid address")
	// ErrRouteNotSupport indicates the device doesn't support routing
	ErrRouteNotSupport = fmt.Errorf("route not support")
	// ErrRecvAborted indicates the receiving is cancelled
	ErrRecvAborted = fmt.Errorf("receiving aborted")
	// ErrRecvEnd indicates the receiving is ended
	ErrRecvEnd = io.EOF
	// ErrAddrNotAvail indicates no more address can be allocated
	ErrAddrNotAvail = fmt.Errorf("address not available")
)

// Runner defines a cancellable background runner
type Runner interface {
	Run(<-chan struct{}) error
}

// MsgReceiver provides a message chan for read
type MsgReceiver interface {
	MsgChan() <-chan prot.Msg
}

// MsgSender writes a message
type MsgSender interface {
	SendMsg(*prot.Msg) error
}

// MsgRouter is able to route a message
type MsgRouter interface {
	RouteMsg(*prot.Msg) error
}

// BusHost is the host side of the bus
type BusHost interface {
	MsgReceiver
	MsgSender
}

// BusSlave is the device side of the bus
type BusSlave interface {
	MsgSender
}

// Bus defines a bus instance
type Bus interface {
	Host() BusHost
	Slave() BusSlave
	Plug(Device) error
	Unplug(Device) error
}

// Device defines a device instance
type Device interface {
	MsgSender
	Address() uint8
	ClassID() uint32
	DeviceID() uint32
	Attach(BusSlave, uint8) error
	Detach() error
}

// Master is the bus master
type Master interface {
	Invoke(addrs []uint8, method uint8, params proto.Message) (Invocation, error)
}

// Invocation represents the result of method invocation
type Invocation interface {
	MsgReceiver
	MessageID() uint32
	Ignore()
}

// MsgReader reads a message from a receiver
type MsgReader struct {
	Timeout    time.Duration
	CancelChan <-chan struct{}
}

// SetTimeout sets the timeout value
func (r *MsgReader) SetTimeout(timeout time.Duration) *MsgReader {
	r.Timeout = timeout
	return r
}

// SetCancelChan sets the cancellation chan
func (r *MsgReader) SetCancelChan(ch <-chan struct{}) *MsgReader {
	r.CancelChan = ch
	return r
}

// ReadMsg reads a message
func (r *MsgReader) ReadMsg(recv MsgReceiver) (*prot.Msg, error) {
	var msg prot.Msg
	var ok bool
	if r.Timeout == 0 {
		if r.CancelChan != nil {
			select {
			case <-r.CancelChan:
				return nil, ErrRecvAborted
			case msg, ok = <-recv.MsgChan():
				break
			}
		} else {
			msg, ok = <-recv.MsgChan()
		}
	} else if r.CancelChan != nil {
		select {
		case <-time.After(r.Timeout):
			return nil, nil
		case _, _ = <-r.CancelChan:
			return nil, ErrRecvAborted
		case msg, ok = <-recv.MsgChan():
			break
		}
	}
	if !ok {
		return nil, ErrRecvEnd
	}
	return &msg, nil
}