package tbus

// Error implements error
func (e *Error) Error() string {
	return e.Message
}
