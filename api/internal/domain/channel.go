package domain

import (
	"regexp"
	"time"
)

const (
	TechFeedChannelID = "tech-feed"
	HeaderChannelID   = "X-Channel-Id"
)

var validChannelIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,49}$`)

func IsValidChannelID(id string) bool {
	return validChannelIDPattern.MatchString(id)
}

type Channel struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
