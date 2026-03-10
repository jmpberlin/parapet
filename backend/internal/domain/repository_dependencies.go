package domain

import "time"

type RepositoryDependency struct {
	ID           string
	RepositoryID string
	Name         string
	Version      string
	PURL         string
	CreatedAt    time.Time
}
