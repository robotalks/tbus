package tbus

// SetOn sets LED on/off state
func (c *LEDCtl) SetOn(on bool) error {
	return c.SetPowerState(&LEDPowerState{On: on})
}

// On is alias for SetOn(true)
func (c *LEDCtl) On() error {
	return c.SetOn(true)
}

// Off is alias for SetOn(false)
func (c *LEDCtl) Off() error {
	return c.SetOn(false)
}
