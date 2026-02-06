package git

import "testing"

func TestAuthorInfo_ContributorKey(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{name: "Lowercase email", email: "user@example.com", expected: "user@example.com"},
		{name: "Uppercase email", email: "USER@EXAMPLE.COM", expected: "user@example.com"},
		{name: "Mixed case email", email: "User@Example.Com", expected: "user@example.com"},
		{name: "Empty email", email: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := AuthorInfo{Name: "Test", Email: tt.email}
			result := a.ContributorKey()
			if result != tt.expected {
				t.Errorf("ContributorKey() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestFileChange_Churn(t *testing.T) {
	tests := []struct {
		name     string
		added    int
		deleted  int
		expected int
	}{
		{name: "Both positive", added: 10, deleted: 5, expected: 15},
		{name: "Only added", added: 10, deleted: 0, expected: 10},
		{name: "Only deleted", added: 0, deleted: 5, expected: 5},
		{name: "Both zero", added: 0, deleted: 0, expected: 0},
		{name: "Large values", added: 1000, deleted: 500, expected: 1500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := FileChange{LinesAdded: tt.added, LinesDeleted: tt.deleted}
			result := f.Churn()
			if result != tt.expected {
				t.Errorf("Churn() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestChangeKind_String(t *testing.T) {
	tests := []struct {
		name     string
		kind     ChangeKind
		expected string
	}{
		{name: "Added", kind: ChangeKindAdded, expected: "added"},
		{name: "Modified", kind: ChangeKindModified, expected: "modified"},
		{name: "Deleted", kind: ChangeKindDeleted, expected: "deleted"},
		{name: "Renamed", kind: ChangeKindRenamed, expected: "renamed"},
		{name: "Unknown", kind: ChangeKind(99), expected: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.kind.String()
			if result != tt.expected {
				t.Errorf("String() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
