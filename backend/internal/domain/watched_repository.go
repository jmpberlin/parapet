package domain

import "time"

type GitProvider string

const (
	Github    GitProvider = "Github.com"
	Bitbucket GitProvider = "Bitbucket.com"
)

type WatchedRepository struct {
	ID             string
	GitProvider    GitProvider
	OwnerName      string
	RepositoryName string
	IntegratedAt   time.Time
	ArchivedAt     *time.Time
}
