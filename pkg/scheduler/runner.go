package scheduler

import (
	"context"
)

func (s *Scheduler) Run(ctx context.Context) error {
	// TODO: wrap this in a timeout
	count, err := s.loadSchedules(ctx)
	if err != nil {
		return err
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
