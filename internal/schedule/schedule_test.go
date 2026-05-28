package schedule

import (
	"testing"
	"time"
)

func TestNextRunDailyLaterToday(t *testing.T) {
	loc := time.FixedZone("KST", 9*60*60)
	from := time.Date(2026, 5, 28, 17, 0, 0, 0, loc)
	got, err := NextRun(Spec{Type: TypeDaily, Time: "18:10"}, from, loc)
	if err != nil {
		t.Fatalf("NextRun() error = %v", err)
	}
	want := time.Date(2026, 5, 28, 18, 10, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("NextRun() = %v, want %v", got, want)
	}
}

func TestNextRunDailyTomorrowAfterTimePassed(t *testing.T) {
	loc := time.FixedZone("KST", 9*60*60)
	from := time.Date(2026, 5, 28, 19, 0, 0, 0, loc)
	got, err := NextRun(Spec{Type: TypeDaily, Time: "18:10"}, from, loc)
	if err != nil {
		t.Fatalf("NextRun() error = %v", err)
	}
	want := time.Date(2026, 5, 29, 18, 10, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("NextRun() = %v, want %v", got, want)
	}
}

func TestNextRunWeeklySelectedWeekday(t *testing.T) {
	loc := time.FixedZone("KST", 9*60*60)
	from := time.Date(2026, 5, 29, 19, 0, 0, 0, loc) // Friday
	got, err := NextRun(Spec{
		Type:     TypeWeekly,
		Time:     "18:10",
		Weekdays: []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
	}, from, loc)
	if err != nil {
		t.Fatalf("NextRun() error = %v", err)
	}
	want := time.Date(2026, 6, 1, 18, 10, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("NextRun() = %v, want %v", got, want)
	}
}

func TestValidateRejectsWeeklyWithoutWeekdays(t *testing.T) {
	if err := Validate(Spec{Type: TypeWeekly, Time: "18:10"}); err == nil {
		t.Fatal("Validate() error = nil for weekly schedule without weekdays")
	}
}
