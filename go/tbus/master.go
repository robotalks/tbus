package tbus

import (
	"container/list"
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

	subs     map[uint8]*pfxMap
	subsLock sync.RWMutex
}

// NewLocalMaster creates a LocalMaster
func NewLocalMaster(dev Device) *LocalMaster {
	m := &LocalMaster{
		Device:            dev,
		InvocationTimeout: DefaultInvocationTimeout,

		invocations: make(map[uint32]*localMasterInvocation),
		subs:        make(map[uint8]*pfxMap),
	}
	dev.AttachTo(m, 0)
	return m
}

// Invoke implements Master
func (m *LocalMaster) Invoke(method uint8, params proto.Message, addrs RouteAddr) Invocation {
	inv := &localMasterInvocation{master: m, timeout: m.InvocationTimeout}

	m.lock.Lock()
	inv.msgID = m.idPool.Alloc()
	if m.invocations == nil {
		m.invocations = make(map[uint32]*localMasterInvocation)
	}
	m.invocations[inv.msgID] = inv
	m.lock.Unlock()

	if inv.err = BuildMsg().
		RouteTo(addrs).
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

// Subscribe implements Master
func (m *LocalMaster) Subscribe(channel uint8, addrs RouteAddr, handler EventHandler) EventSubscription {
	m.subsLock.Lock()
	defer m.subsLock.Unlock()
	subsMap := m.subs[channel]
	if subsMap == nil {
		subsMap = newPfxMap()
		m.subs[channel] = subsMap
	}
	subs := &subscribers{master: m, channel: channel, addrs: addrs}
	if exist := subsMap.insert(addrs, subs); exist != nil {
		subs = exist.(*subscribers)
	}
	return subs.add(handler)
}

// Unsubscribe removes a subscription
func (m *LocalMaster) Unsubscribe(sub EventSubscription) error {
	subscription := sub.(*subscription)
	subscribers := subscription.owner
	if subscribers == nil {
		return nil
	}
	m.subsLock.Lock()
	defer m.subsLock.Unlock()
	// check again for concurrent Unsubscribe with same subscription
	if subscribers = subscription.owner; subscribers == nil {
		return nil
	}
	subscribers.remove(subscription)
	if subscribers.empty() {
		chnMap := m.subs[subscribers.channel]
		if chnMap != nil {
			chnMap.remove(subscribers.addrs)
			if chnMap.empty() {
				delete(m.subs, subscribers.channel)
			}
		}
	}
	return nil
}

// DispatchMsg implements BusPort
func (m *LocalMaster) DispatchMsg(msg *Msg) error {
	if msg.Head.IsEvent() {
		return m.dispatchEvent(msg)
	}
	go m.recvReply(*msg)
	return nil
}

func (m *LocalMaster) dispatchEvent(msg *Msg) error {
	m.subsLock.RLock()
	defer m.subsLock.RUnlock()
	subsMap := m.subs[msg.Body.Flag]
	if subsMap == nil {
		return nil
	}
	val := subsMap.lookup(msg.Head.Addrs)
	if val == nil {
		return nil
	}
	val.(*subscribers).emit(msg)
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

type subscribers struct {
	master  *LocalMaster
	channel uint8
	addrs   RouteAddr
	subs    list.List
}

func (s *subscribers) add(handler EventHandler) *subscription {
	sub := &subscription{owner: s, handler: handler}
	sub.elem = s.subs.PushBack(sub)
	return sub
}

func (s *subscribers) remove(sub *subscription) {
	if sub.owner != s {
		panic("inconsistent subscription")
	}
	if sub.elem != nil {
		s.subs.Remove(sub.elem)
		sub.elem = nil
	}
	sub.owner = nil
}

func (s *subscribers) empty() bool {
	return s.subs.Len() == 0
}

func (s *subscribers) emit(msg *Msg) {
	for elem := s.subs.Front(); elem != nil; elem = elem.Next() {
		elem.Value.(*subscription).emit(msg)
	}
}

type subscription struct {
	owner   *subscribers
	handler EventHandler
	elem    *list.Element
}

func (s *subscription) emit(msg *Msg) {
	evt := &subscribedEvent{msg: *msg}
	go s.handler.HandleEvent(evt, s)
}

func (s *subscription) Close() error {
	return s.owner.master.Unsubscribe(s)
}

type subscribedEvent struct {
	msg Msg
}

func (e *subscribedEvent) Channel() uint8 {
	return e.msg.Body.Flag
}

func (e *subscribedEvent) Address() RouteAddr {
	return e.msg.Head.Addrs
}

func (e *subscribedEvent) Decode(out proto.Message) error {
	return e.msg.Body.Decode(out)
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
