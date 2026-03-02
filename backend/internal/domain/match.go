package domain

import "time"

type MatchStatus string

const (
	MatchStatusNew      MatchStatus = "NEW"
	MatchStatusAlerting MatchStatus = "ALERTING"
	MatchStatusResolved MatchStatus = "RESOLVED"
)

type Match struct {
	ID               string
	VulnerabilityID  string
	RepositoryID     string
	MatchedComponent string
	MatchedVersion   string
	Status           MatchStatus
	CreatedAt        time.Time
}
