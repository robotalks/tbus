package tbus

import "time"

// InvokeMotorSetSpeed represents the invocation of Motor.SetSpeed
type InvokeMotorSetSpeed struct {
	MethodInvocation
}

// Timeout implements Invocation
func (i *InvokeMotorSetSpeed) Timeout(dur time.Duration) *InvokeMotorSetSpeed {
	i.Invocation.Timeout(dur)
	return i
}

// Wait waits and retrieves the result
func (i *InvokeMotorSetSpeed) Wait() error {
	return i.Result(nil)
}

// Forward starts the motor forward
func (c *MotorCtl) Forward(speed int) *InvokeMotorSetSpeed {
	return c.SetSpeed(speed)
}

// Reverse starts the motor reverse
func (c *MotorCtl) Reverse(speed int) *InvokeMotorSetSpeed {
	return c.SetSpeed(-speed)
}

// SetSpeed starts/stops the motor:
// - forward: when speed > 0
// - reverse: when speed < 0
// - stop: when speed = 0
func (c *MotorCtl) SetSpeed(speed int) *InvokeMotorSetSpeed {
	invoke := &InvokeMotorSetSpeed{}
	if speed == 0 {
		invoke.Invocation = c.Stop().Invocation
	} else {
		dir := MotorDriveState_Fwd
		if speed < 0 {
			dir = MotorDriveState_Rev
			speed = -speed
		}
		invoke.Invocation = c.Start(&MotorDriveState{
			Direction: dir,
			Speed:     uint32(speed),
		}).Invocation
	}
	return invoke
}

// SetBrake sets brake on/off
func (c *MotorCtl) SetBrake(on bool) *InvokeMotorBrake {
	return c.Brake(&MotorBrakeState{On: on})
}
