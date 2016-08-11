package tbus

import (
	prot "github.com/evo-bots/tbus/go/tbus/protocol"
	proto "github.com/golang/protobuf/proto"
)

// Controller is the base class of device controller
type Controller struct {
	MsgReader
	Master  Master
	Address []uint8
}

// Invoke invokes a method on a device
func (c *Controller) Invoke(methodIndex uint8, params proto.Message) (*prot.Msg, error) {
	invocation, err := c.Master.Invoke(c.Address, methodIndex, params)
	if err != nil {
		return nil, err
	}
	return c.ReadMsg(invocation)
}
