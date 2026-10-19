package dfplayer

// ErrUartRead represents an error that occurred while reading from the UART.
type ErrUartRead struct {
	text string
	err  error
}

// Error returns the error message.
func (e *ErrUartRead) Error() string {
	return e.text
}

// Unwrap returns the underlying error for errors.Unwrap support.
func (e *ErrUartRead) Unwrap() error {
	return e.err
}

// ErrUartWrite represents an error that occurred while writing to the UART.
type ErrUartWrite struct {
	text string
	err  error
}

// Error returns the error message.
func (e *ErrUartWrite) Error() string {
	return e.text
}

// Unwrap returns the underlying error for errors.Unwrap support.
func (e *ErrUartWrite) Unwrap() error {
	return e.err
}

// ErrDfplayer represents a DFPlayer-specific error (invalid parameters,
// checksum errors, etc.).
type ErrDfplayer struct {
	text string
}

// Error returns the error message.
func (e *ErrDfplayer) Error() string {
	return e.text
}