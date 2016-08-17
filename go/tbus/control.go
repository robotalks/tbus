package tbus

import proto "github.com/golang/protobuf/proto"

// Controller is the base class of device controller
type Controller struct {
	LogicBase
	MsgReader
	Master  Master
	Address []uint8
}

// Invoke invokes a method on a device
func (c *Controller) Invoke(methodIndex uint8, params proto.Message, reply proto.Message) error {
	invocation, err := c.Master.Invoke(methodIndex, params, c.Address)
	if err != nil {
		return err
	}
	return c.ReadReply(invocation, reply)
}
