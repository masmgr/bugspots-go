package output

import (
	"testing"
	"time"
)

func TestLimitTop(t *testing.T) {
	items := []int{1, 2, 3}

	tests := []struct {
		name string
		top  int
		want []int
	}{
		{name: "NoLimitWhenZero", top: 0, want: []int{1, 2, 3}},
		{name: "NoLimitWhenNegative", top: -1, want: []int{1, 2, 3}},
		{name: "Limited", top: 2, want: []int{1, 2}},
		{name: "NoLimitWhenTopExceedsLength", top: 5, want: []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := limitTop(items, tt.top)
			if len(got) != len(tt.want) {
				t.Fatalf("len(limitTop(..., %d)) = %d, want %d", tt.top, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("limitTop(..., %d)[%d] = %d, want %d", tt.top, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDateRangeLabelAndValue(t *testing.T) {
	until := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	t.Run("WithSince", func(t *testing.T) {
		since := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		label, value := dateRangeLabelAndValue(&since, until)
		if label != "Period" {
			t.Fatalf("label = %q, want %q", label, "Period")
		}
		if value != "2026-02-01 to 2026-02-10" {
			t.Fatalf("value = %q, want %q", value, "2026-02-01 to 2026-02-10")
		}
	})

	t.Run("WithoutSince", func(t *testing.T) {
		label, value := dateRangeLabelAndValue(nil, until)
		if label != "Until" {
			t.Fatalf("label = %q, want %q", label, "Until")
		}
		if value != "2026-02-10" {
			t.Fatalf("value = %q, want %q", value, "2026-02-10")
		}
	})
}

func TestFormatSinceDate(t *testing.T) {
	if got := formatSinceDate(nil); got != nil {
		t.Fatalf("formatSinceDate(nil) = %v, want nil", got)
	}

	since := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	got := formatSinceDate(&since)
	if got == nil || *got != "2026-02-01" {
		t.Fatalf("formatSinceDate(...) = %v, want %q", got, "2026-02-01")
	}
}
