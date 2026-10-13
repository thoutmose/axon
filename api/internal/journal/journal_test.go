package journal

import (
	"strings"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestAnEntryIsChecked(t *testing.T) {
	ok := Fields{Mood: ptr(4), Energy: ptr(3), Stress: ptr(2), SleepHours: ptr(7.5),
		Gratitude: []string{" Coffee with Sam ", "", "Sunny walk"}, Note: "A good day."}
	if err := ok.check(); err != nil {
		t.Fatal(err)
	}
	if len(ok.Gratitude) != 2 || ok.Gratitude[0] != "Coffee with Sam" {
		t.Fatalf("gratitude = %q", ok.Gratitude)
	}
	for name, bad := range map[string]Fields{
		"empty":        {},
		"mood 0":       {Mood: ptr(0)},
		"energy 6":     {Energy: ptr(6)},
		"stress 9":     {Stress: ptr(9)},
		"sleep 25":     {SleepHours: ptr(25.0)},
		"sleep -1":     {SleepHours: ptr(-1.0)},
		"4 gratitudes": {Gratitude: []string{"a", "b", "c", "d"}},
		"long thanks":  {Gratitude: []string{strings.Repeat("x", 301)}},
		"long note":    {Note: strings.Repeat("x", 10_001)},
	} {
		if bad.check() == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
