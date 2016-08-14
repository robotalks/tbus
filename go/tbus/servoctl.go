package tbus

// MoveTo is wrapper for SetPosition
func (c *ServoCtl) MoveTo(angle int) error {
	return c.SetPosition(&ServoPosition{Angle: uint32(angle)})
}
