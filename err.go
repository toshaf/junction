package junction

import (
	"fmt"
)

type Error string

func Err(format string, args ...interface{}) error {
	return Error(fmt.Sprintf(format, args...))
}

func (e Error) Error() string {
	return string(e)
}
