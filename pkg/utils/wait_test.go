package utils

import (
	"errors"
	"testing"
	"time"
)

func TestRetry_ok(t *testing.T) {
	i := 0
	err := Retry(1*time.Millisecond, 3*time.Millisecond, func() error {
		i = i + 1
		return nil
	})
	if err != nil {
		t.Errorf("error should be nil")
	}
	if i != 1 {
		t.Errorf("function should be only called once")
	}
}

func TestRetry_timeout(t *testing.T) {
	i := 0
	err := Retry(1*time.Millisecond, 3*time.Millisecond, func() error {
		i = i + 1
		return errors.New("error")
	})
	if err == nil {
		t.Errorf("error should not be nil")
	}
	if i < 2 {
		t.Errorf("function should be called at least twice")
	}
	if err.Error() != "timeout" {
		t.Errorf("it should be timed out")
	}
}
