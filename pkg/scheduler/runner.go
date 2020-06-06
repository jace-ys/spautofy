package scheduler

import (
	"context"
	"fmt"
	"time"
)

func (s *Scheduler) Run(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	var count int
	var err error
	for {
		count, err = s.loadSchedules(ctx)
		if err == nil {
			break
		}

		s.logger.Log("event", "schedules.load.retried")
		select {
		case <-time.After(15 * time.Second):
			continue
		case <-ctx.Done():
			return fmt.Errorf("%w: %s", ctx.Err(), err)
		}
	}

	s.logger.Log("event", "schedules.loaded", "scheduled", count)
	s.runner.Run()

	return nil
}

func (s *Scheduler) loadSchedules(ctx context.Context) (int, error) {
	schedules, err := s.List(ctx)
	if err != nil {
		return 0, err
	}

	for idx, schedule := range schedules {
		_, err := s.Create(ctx, schedule.UserID, schedule.Spec, schedule.WithEmail)
		if err != nil {
			return idx, err
		}
	}

	return len(schedules), nil
}

func (s *Scheduler) Stop() error {
	s.runner.Stop()
	return nil
}
