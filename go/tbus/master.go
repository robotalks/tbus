package tbus

import (
	"sync"

	prot "github.com/evo-bots/tbus/go/tbus/protocol"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
)

// LocalMaster implements master for local controllers
type LocalMaster struct {
	BusPort BusHostPort

	idPool      MinIDGen
	invocations map[uint32]*localMasterInvocation
	lock        sync.Mutex
}

// NewLocalMaster creates a LocalMaster
func NewLocalMaster(bus BusHostPort) *LocalMaster {
	return &LocalMaster{
		BusPort:     bus,
		invocations: make(map[uint32]*localMasterInvocation),
	}
}

// Invoke implements Master
func (m *LocalMaster) Invoke(addrs []uint8, method uint8, params proto.Message) (Invocation, error) {
	if params == nil {
		params = &empty.Empty{}
	}
	body, err := proto.Marshal(params)
	if err != nil {
		return nil, err
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	msgID := m.allocID()
	msg, err := prot.EncodeAsMsg(addrs, msgID, method, body)
	if err != nil {
		m.releaseID(msgID)
		return nil, err
	}

	err = m.BusPort.SendMsg(msg)
	if err != nil {
		m.releaseID(msgID)
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

// Run implements Runner
func (m *LocalMaster) Run(stopCh <-chan struct{}) error {
	reader := &MsgReader{CancelChan: stopCh}
	for {
		msg, err := reader.ReadMsg(m.BusPort)
		if err == ErrRecvAborted || (err == nil && msg == nil) {
			return err
		}
		if err == nil {
			m.lock.Lock()
			inv := m.invocations[msg.Head.MsgID]
			if inv != nil {
				delete(m.invocations, msg.Head.MsgID)
				m.releaseID(inv.msgID)
			}
			m.lock.Unlock()
			if inv != nil {
				inv.replyCh <- *msg
			}
		}
		// else TODO log error
	}
}

func (m *LocalMaster) allocID() uint32 {
	return m.idPool.Alloc()
}

func (m *LocalMaster) releaseID(id uint32) {
	m.idPool.Release(id)
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
		}
		c.master.lock.Unlock()
	}
}
