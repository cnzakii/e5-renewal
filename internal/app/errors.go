package app

import "errors"

// Error codes are the client contract; English messages are diagnostic fallbacks.
type codedError struct {
	code    string
	message string
}

func newError(code, message string) *codedError { return &codedError{code, message} }
func (e *codedError) Error() string             { return e.message }
func (e *codedError) ErrorCode() string         { return e.code }

func errorCode(err error) string {
	var coded interface{ ErrorCode() string }
	if errors.As(err, &coded) {
		return coded.ErrorCode()
	}

	return "internal_error"
}
