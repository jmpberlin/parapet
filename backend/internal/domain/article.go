package domain

import "time"

type HostDomain string

const (
	HackerNews       HostDomain = "hackernews.com"
	BleepingComputer HostDomain = "bleepingcomputer.com"
)

type Article struct {
	ID             string
	SourceURL      string
	HostDomain     HostDomain
	PublishedAt    time.Time
	Headline       string
	Author         string
	ContentHTML    string
	ContentCleaned string
	CrawledAt      time.Time
}
