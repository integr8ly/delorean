package utils

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestParallelLimit(t *testing.T) {
	cases := []struct {
		description string
		limit       int
		taskFactory func() []Task
		expectError bool
		checkResult func([]TaskResult) error
	}{
		{
			description: "all tasks should be finished",
			limit:       2,
			expectError: false,
			taskFactory: func() []Task {
				numberOfTasks := 4
				tasks := make([]Task, numberOfTasks)
				for i := 0; i < numberOfTasks; i++ {
					v := i
					t := func() (TaskResult, error) {
						return v, nil
					}
					tasks[i] = t
				}
				return tasks
			},
			checkResult: func(results []TaskResult) error {
				for i, v := range results {
					val := v.(int)
					if i != val {
						return fmt.Errorf("expected %d but got %d", i, val)
					}
				}
				return nil
			},
		},
		{
			description: "should return error when any task returns error",
			limit:       2,
			expectError: true,
			taskFactory: func() []Task {
				return []Task{
					func() (result TaskResult, err error) {
						return 1, nil
					},
					func() (result TaskResult, err error) {
						return nil, errors.New("test error")
					},
				}
			},
			checkResult: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			tasks := c.taskFactory()
			results, err := ParallelLimit(context.Background(), tasks, c.limit)
			if err != nil && !c.expectError {
				t.Fatalf("unexpected error: %v", err)
			} else if c.expectError && err == nil {
				t.Fatal("expect error but got nil")
			}

			if c.checkResult != nil {
				if err := c.checkResult(results); err != nil {
					t.Fatalf("checkResult failed with error: %v", err)
				}
			}
		})
	}
}
