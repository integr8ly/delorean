package utils

import (
	"context"
	"golang.org/x/sync/errgroup"
)

type TaskResult interface{}

type Task func() (TaskResult, error)

type taskWithId struct {
	id int
	t  Task
}

type resultWithId struct {
	id int
	r  TaskResult
}

// Run the given number of tasks in parallel with the given concurrency limit.
// It will wait for all tasks to be completed, and return either the first error if there's any, or the results
func ParallelLimit(ctx context.Context, tasks []Task, limit int) ([]TaskResult, error) {
	numberOfTasks := len(tasks)
	g, ctx := errgroup.WithContext(ctx)
	tasksChan := make(chan taskWithId, numberOfTasks)
	resultsChan := make(chan resultWithId, numberOfTasks)
	workers := Min(limit, numberOfTasks)
	for i := 0; i < workers; i++ {
		g.Go(func() error {
			for t := range tasksChan {
				r, err := t.t()
				if err != nil {
					return err
				}
				select {
				case resultsChan <- resultWithId{
					id: t.id,
					r:  r,
				}:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}
	for i, t := range tasks {
		idt := taskWithId{
			id: i,
			t:  t,
		}
		tasksChan <- idt
	}
	close(tasksChan)

	go func() {
		g.Wait()
		//make sure the channel is closed so that we can loop through the results
		close(resultsChan)
	}()

	results := make([]TaskResult, numberOfTasks)
	for r := range resultsChan {
		results[r.id] = r.r
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
