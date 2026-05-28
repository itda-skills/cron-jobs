package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	TypeDaily  = "daily"
	TypeWeekly = "weekly"
)

type Spec struct {
	Type     string   `json:"type"`
	Time     string   `json:"time"`
	Weekdays []string `json:"weekdays,omitempty"`
}

func Validate(spec Spec) error {
	if spec.Type != TypeDaily && spec.Type != TypeWeekly {
		return fmt.Errorf("unsupported schedule type %q", spec.Type)
	}
	if _, _, err := ParseClock(spec.Time); err != nil {
		return err
	}
	if spec.Type == TypeDaily {
		if len(spec.Weekdays) != 0 {
			return fmt.Errorf("daily schedule must not define weekdays")
		}
		return nil
	}
	if len(spec.Weekdays) == 0 {
		return fmt.Errorf("weekly schedule requires at least one weekday")
	}
	seen := map[time.Weekday]struct{}{}
	for _, raw := range spec.Weekdays {
		weekday, err := ParseWeekday(raw)
		if err != nil {
			return err
		}
		if _, ok := seen[weekday]; ok {
			return fmt.Errorf("duplicate weekday %q", raw)
		}
		seen[weekday] = struct{}{}
	}
	return nil
}

func NextRun(spec Spec, from time.Time, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.Local
	}
	if err := Validate(spec); err != nil {
		return time.Time{}, err
	}

	from = from.In(loc)
	hour, minute, _ := ParseClock(spec.Time)
	if spec.Type == TypeDaily {
		candidate := time.Date(from.Year(), from.Month(), from.Day(), hour, minute, 0, 0, loc)
		if candidate.After(from) {
			return candidate, nil
		}
		return candidate.AddDate(0, 0, 1), nil
	}

	selected := map[time.Weekday]struct{}{}
	for _, raw := range spec.Weekdays {
		weekday, _ := ParseWeekday(raw)
		selected[weekday] = struct{}{}
	}
	for offset := 0; offset <= 7; offset++ {
		day := from.AddDate(0, 0, offset)
		if _, ok := selected[day.Weekday()]; !ok {
			continue
		}
		candidate := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc)
		if candidate.After(from) {
			return candidate, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not compute next weekly run")
}

func ParseClock(value string) (int, int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 || len(parts[0]) != 2 || len(parts[1]) != 2 {
		return 0, 0, fmt.Errorf("time %q must use HH:MM format", value)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("time %q has invalid hour", value)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("time %q has invalid minute", value)
	}
	return hour, minute, nil
}

func ParseWeekday(value string) (time.Weekday, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "sunday", "sun":
		return time.Sunday, nil
	case "monday", "mon":
		return time.Monday, nil
	case "tuesday", "tue":
		return time.Tuesday, nil
	case "wednesday", "wed":
		return time.Wednesday, nil
	case "thursday", "thu":
		return time.Thursday, nil
	case "friday", "fri":
		return time.Friday, nil
	case "saturday", "sat":
		return time.Saturday, nil
	default:
		return time.Sunday, fmt.Errorf("invalid weekday %q", value)
	}
}
