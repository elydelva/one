package portstest

import "errors"

func isErrorAs(err error, target any) bool {
	return errors.As(err, target)
}
