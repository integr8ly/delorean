package utils

import (
	"errors"
	"time"
)

// Retry can be used to execute the given function f with the given interval, until the given timeout value is exceeded.
func Retry(interval time.Duration, timeout time.Duration, f func() (bool, error)) error {
	done := make(chan bool)
	var err error
	go func() {
		for {
			time.Sleep(interval)
			ok, er := f()
			if ok {
				err = er
				done <- true
			}
		}
	}()
	for {
		select {
		case <-done:
			return err
		case <-time.After(timeout):
			return errors.New("timeout")
		}
	}
}
