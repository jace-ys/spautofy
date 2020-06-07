package scheduler

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
)

var (
	specRe = regexp.MustCompile(`0 0 1 1\/([1-9]+) *`)
)

type Schedule struct {
	ID        cron.EntryID
	UserID    string
	Spec      string
	Limit     int // TODO: allow custom limit
	WithEmail bool
	Cmd       cron.FuncJob
	CreatedAt time.Time
}

func NewSchedule(userID, spec string, withEmail bool, cmd cron.FuncJob) *Schedule {
	return &Schedule{
		UserID:    userID,
		Spec:      spec,
		WithEmail: withEmail,
		Cmd:       cmd,
	}
}

func (s *Schedule) Frequency() int {
	return SpecToFrequency(s.Spec)
}

func (s *Schedule) Next() time.Time {
	return GetNext(s.Spec)
}

func SpecToFrequency(spec string) int {
	match := specRe.FindStringSubmatch(spec)
	if len(match) < 2 {
		return 0
	}

	step, _ := strconv.Atoi(match[1])
	return 12 / step
}

func FrequencyToSpec(frequency int) string {
	step := 12 / frequency
	return fmt.Sprintf("0 0 1 1/%d *", step)
}

func GetNext(spec string) time.Time {
	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(time.Now())
}
