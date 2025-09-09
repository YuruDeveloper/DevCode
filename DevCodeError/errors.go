package devcodeerror

import (
	"fmt"
	"time"
)

type DevCodeError struct {
	ErrorCode ErrorCode
	Message   string
	Cause     error
	Timestap  time.Time
}

func (instance *DevCodeError) Error() string {
	if instance.Cause != nil {
		return fmt.Sprintf("[%d] %s : %v", instance.ErrorCode, instance.Message, instance.Cause)
	}
	return fmt.Sprintf("[%d] %s", instance.ErrorCode, instance.Message)
}

func Wrap(err error, errorCode ErrorCode, message string) *DevCodeError {
	return &DevCodeError{
		ErrorCode: errorCode,
		Message:   message,
		Cause:     err,
		Timestap:  time.Now(),
	}
}
