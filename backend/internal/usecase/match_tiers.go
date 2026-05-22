package usecase

type MatchConfidence string

const (
	ConfidenceHigh      MatchConfidence = "HIGH"
	ConfidenceMedium    MatchConfidence = "MEDIUM"
	ConfidenceLow       MatchConfidence = "LOW"
	ConfidenceWarning   MatchConfidence = "WARNING"
	ConfidenceConfirmed MatchConfidence = "CONFIRMED"
	ConfidenceNone      MatchConfidence = "NONE"
)

type tierResult struct {
	matched        bool
	confidence     MatchConfidence
	matchedOn      string
	vulnIdentifier string
	depIdentifier  string
}
