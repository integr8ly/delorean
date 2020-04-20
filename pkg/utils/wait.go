package utils

import (
	"errors"
	"time"
)

// Retry can be used to execute the given function f with the given interval, until the given timeout value is exceeded.
func Retry(interval time.Duration, timeout time.Duration, f func() error) error {
	done := make(chan bool)
	go func() {
		for {
			time.Sleep(interval)
			err := f()
			if err == nil {
				done <- true
			}
		}
	}()
	for {
		select {
		case <-done:
			return nil
		case <-time.After(timeout):
			return errors.New("timeout")
		}
	}
}
