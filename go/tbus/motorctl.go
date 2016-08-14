package tbus

// Forward starts the motor forward
func (c *MotorCtl) Forward(speed int) error {
	return c.SetSpeed(speed)
}

// Reverse starts the motor reverse
func (c *MotorCtl) Reverse(speed int) error {
	return c.SetSpeed(-speed)
}

// SetSpeed starts/stops the motor:
// - forward: when speed > 0
// - reverse: when speed < 0
// - stop: when speed = 0
func (c *MotorCtl) SetSpeed(speed int) error {
	if speed == 0 {
		return c.Stop()
	}
	dir := MotorDriveState_Fwd
	if speed < 0 {
		dir = MotorDriveState_Rev
		speed = -speed
	}
	return c.Start(&MotorDriveState{
		Direction: dir,
		Speed:     uint32(speed),
	})
}

// SetBrake sets brake on/off
func (c *MotorCtl) SetBrake(on bool) error {
	return c.Brake(&MotorBrakeState{On: on})
}
