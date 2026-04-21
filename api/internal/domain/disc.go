package domain

import "time"

type Disc struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	DiscName  *string   `json:"disc_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
