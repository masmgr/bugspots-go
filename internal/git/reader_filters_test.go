package git

import "testing"

func TestHistoryReader_matchesFilters_InvalidPatternsReturnError(t *testing.T) {
	t.Run("invalid exclude pattern", func(t *testing.T) {
		r := &HistoryReader{
			opts:        ReadOptions{Exclude: []string{"["}},
			filterCache: make(map[string]bool),
		}
		_, err := r.matchesFilters("a.go")
		if err == nil {
			t.Fatal("expected error for invalid exclude glob, got nil")
		}
	})

	t.Run("invalid include pattern", func(t *testing.T) {
		r := &HistoryReader{
			opts:        ReadOptions{Include: []string{"["}},
			filterCache: make(map[string]bool),
		}
		_, err := r.matchesFilters("a.go")
		if err == nil {
			t.Fatal("expected error for invalid include glob, got nil")
		}
	})
}
