package scheduler

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/robfig/cron/v3"
)

var (
	specRe = regexp.MustCompile(`0 0 1 1\/([1-9]+) *`)
)

type Schedule struct {
	ID        cron.EntryID
	UserID    string
	Spec      string
	WithEmail bool
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
