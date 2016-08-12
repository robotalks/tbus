package tbus

import (
	"sync"

	prot "github.com/evo-bots/tbus/go/tbus/protocol"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
)

// LocalMaster implements master for local controllers
type LocalMaster struct {
	Device Device

	idPool      MinIDGen
	invocations map[uint32]*localMasterInvocation
	lock        sync.Mutex
}

// NewLocalMaster creates a LocalMaster
func NewLocalMaster(dev Device) *LocalMaster {
	m := &LocalMaster{
		Device:      dev,
		invocations: make(map[uint32]*localMasterInvocation),
	}
	dev.AttachTo(m, 0)
	return m
}

// Invoke implements Master
func (m *LocalMaster) Invoke(method uint8, params proto.Message, addrs []uint8) (Invocation, error) {
	if params == nil {
		params = &empty.Empty{}
	}
	body, err := proto.Marshal(params)
	if err != nil {
		return nil, err
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	msgID := m.idPool.Alloc()
	msg, err := prot.EncodeAsMsg(addrs, msgID, method, body)
	if err != nil {
		m.idPool.Release(msgID)
		return nil, err
	}

	err = m.Device.SendMsg(msg)
	if err != nil {
		m.idPool.Release(msgID)
		return nil, err
	}

	inv := &localMasterInvocation{
		master:  m,
		msgID:   msg.Head.MsgID,
		replyCh: make(chan prot.Msg, 1),
	}

	if m.invocations == nil {
		m.invocations = make(map[uint32]*localMasterInvocation)
	}
	m.invocations[inv.msgID] = inv
	return inv, nil
}

// SendMsg implements BusPort
func (m *LocalMaster) SendMsg(msg *prot.Msg) error {
	go m.recvReply(*msg)
	return nil
}

func (m *LocalMaster) recvReply(msg prot.Msg) {
	m.lock.Lock()
	inv := m.invocations[msg.Head.MsgID]
	if inv != nil {
		delete(m.invocations, msg.Head.MsgID)
		m.idPool.Release(inv.msgID)
	}
	m.lock.Unlock()
	if inv != nil {
		inv.replyCh <- msg
	}
}

type localMasterInvocation struct {
	master  *LocalMaster
	msgID   uint32
	replyCh chan prot.Msg
}

func (c *localMasterInvocation) MsgChan() <-chan prot.Msg {
	return c.replyCh
}

func (c *localMasterInvocation) MessageID() uint32 {
	return c.msgID
}

func (c *localMasterInvocation) Ignore() {
	if c.master != nil {
		c.master.lock.Lock()
		if c.master.invocations != nil {
			delete(c.master.invocations, c.msgID)
			c.master.idPool.Release(c.msgID)
		}
		c.master.lock.Unlock()
	}
}
