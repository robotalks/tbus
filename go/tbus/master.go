package tbus

import (
	"sync"
	"time"

	proto "github.com/golang/protobuf/proto"
)

const (
	// DefaultInvocationTimeout specifies the default invocation timeout
	DefaultInvocationTimeout = time.Minute
)

// LocalMaster implements master for local controllers
type LocalMaster struct {
	Device            Device
	InvocationTimeout time.Duration

	idPool      MinIDGen
	invocations map[uint32]*localMasterInvocation
	lock        sync.Mutex
}

// NewLocalMaster creates a LocalMaster
func NewLocalMaster(dev Device) *LocalMaster {
	m := &LocalMaster{
		Device:            dev,
		InvocationTimeout: DefaultInvocationTimeout,

		invocations: make(map[uint32]*localMasterInvocation),
	}
	dev.AttachTo(m, 0)
	return m
}

// Invoke implements Master
func (m *LocalMaster) Invoke(method uint8, params proto.Message, addrs []uint8) Invocation {
	inv := &localMasterInvocation{master: m, timeout: m.InvocationTimeout}

	m.lock.Lock()
	inv.msgID = m.idPool.Alloc()
	if m.invocations == nil {
		m.invocations = make(map[uint32]*localMasterInvocation)
	}
	m.invocations[inv.msgID] = inv
	m.lock.Unlock()

	if inv.err = BuildMsg().
		RouteTo(addrs...).
		MsgIDVarInt(inv.msgID).
		EncodeBody(method, params).
		Build().
		Dispatch(m.Device); inv.err != nil {
		inv.release()
	} else {
		inv.replyCh = make(chan Msg, 1)
	}

	return inv
}

// DispatchMsg implements BusPort
func (m *LocalMaster) DispatchMsg(msg *Msg) error {
	go m.recvReply(*msg)
	return nil
}

func (m *LocalMaster) recvReply(msg Msg) {
	msgID, err := msg.Head.MsgID.VarInt()
	if err != nil {
		// discard improper message
		return
	}
	m.lock.Lock()
	inv := m.invocations[msgID]
	if inv != nil {
		delete(m.invocations, msgID)
		m.idPool.Release(msgID)
	}
	m.lock.Unlock()
	if inv != nil && inv.replyCh != nil {
		inv.replyCh <- msg
	}
}

type localMasterInvocation struct {
	err     error
	master  *LocalMaster
	msgID   uint32
	replyCh chan Msg
	timeout time.Duration
}

func (c *localMasterInvocation) Recv() (MsgReceiver, error) {
	return c, c.err
}

func (c *localMasterInvocation) MsgChan() <-chan Msg {
	return c.replyCh
}

func (c *localMasterInvocation) MsgID() MsgID {
	return MsgIDVarInt(c.msgID)
}

func (c *localMasterInvocation) Timeout(dur time.Duration) Invocation {
	c.timeout = dur
	return c
}

func (c *localMasterInvocation) Result(reply proto.Message) error {
	recv, err := c.Recv()
	if err != nil {
		return err
	}
	if recv == nil {
		return ErrRecvEnd
	}
	var msg Msg
	var ok bool
	if c.timeout == 0 {
		msg, ok = <-recv.MsgChan()
	} else {
		select {
		case <-time.After(c.timeout):
			return ErrRecvTimeout
		case msg, ok = <-recv.MsgChan():
			break
		}
	}
	if !ok {
		return ErrRecvEnd
	}

	return msg.Body.Decode(reply)
}

func (c *localMasterInvocation) Ignore() {
	c.release()
}

func (c *localMasterInvocation) release() {
	if c.master != nil {
		c.master.lock.Lock()
		if c.master.invocations != nil {
			delete(c.master.invocations, c.msgID)
			c.master.idPool.Release(c.msgID)
		}
		c.master.lock.Unlock()
	}
}
