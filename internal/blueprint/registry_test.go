package blueprint

import (
	"testing"
)

func TestRegistry_RegisterAndSearch(t *testing.T) {
	bp := &Blueprint{
		ID:          "resend-mailer",
		Version:     "1.0.0",
		Title:       "Resend Transactional Mailer",
		Description: "Send transactional emails and tokens",
		Category:    "Communications",
		Tags:        []string{"email", "resend", "transactional"},
		Target: TargetSpec{
			Language:  "go",
			Framework: "fiber",
		},
		Env: []EnvVar{
			{Key: "RESEND_API_KEY", Required: true, Description: "API key"},
		},
	}

	Register(bp)

	retrieved, err := Get("resend-mailer")
	if err != nil {
		t.Fatalf("expected Get to find resend-mailer: %v", err)
	}
	if retrieved.Title != bp.Title {
		t.Errorf("expected title %q, got %q", bp.Title, retrieved.Title)
	}

	// Search by tag
	matches := Search("", "", "resend")
	if len(matches) == 0 {
		t.Errorf("expected search for tag 'resend' to find at least 1 match")
	}

	// Search by query
	matchesQuery := Search("transactional", "", "")
	if len(matchesQuery) == 0 {
		t.Errorf("expected search for query 'transactional' to find at least 1 match")
	}

	// Search by category
	matchesCategory := Search("", "Communications", "")
	if len(matchesCategory) == 0 {
		t.Errorf("expected search for category 'Communications' to find at least 1 match")
	}
}
