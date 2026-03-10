package domain

import "time"

type MatchStatus string

const (
	MatchStatusConfirmed MatchStatus = "CONFIRMED"
	MatchStatusWarning   MatchStatus = "WARNING"
	MatchStatusResolved  MatchStatus = "RESOLVED"
)

type Match struct {
	ID               string
	VulnerabilityID  string
	RepositoryID     string
	ComponentPURL    string
	MatchedComponent string
	MatchedVersion   string
	Status           MatchStatus
	ResolvedAt       *time.Time
	CreatedAt        time.Time
}
