package model

import "time"

type Link struct {
	ID          int64
	URL         string
	Tags        []string
	ChatIDs     []int64
	LastUpdated time.Time
}
