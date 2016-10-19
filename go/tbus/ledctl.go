package tbus

// SetOn sets LED on/off state
func (c *LEDCtl) SetOn(on bool) *InvokeLEDSetPowerState {
	return c.SetPowerState(&LEDPowerState{On: on})
}

// On is alias for SetOn(true)
func (c *LEDCtl) On() *InvokeLEDSetPowerState {
	return c.SetOn(true)
}

// Off is alias for SetOn(false)
func (c *LEDCtl) Off() *InvokeLEDSetPowerState {
	return c.SetOn(false)
}
