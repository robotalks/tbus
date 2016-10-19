package tbus

// MoveTo is wrapper for SetPosition
func (c *ServoCtl) MoveTo(angle int) *InvokeServoSetPosition {
	return c.SetPosition(&ServoPosition{Angle: uint32(angle)})
}
