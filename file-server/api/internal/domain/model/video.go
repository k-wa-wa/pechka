package model

import "time"

type VideoEntity struct {
	Id          string     `json:"id"`
	Fullpath    string     `json:"-"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Url         string     `json:"url"`
	CreatedAt   *time.Time `json:"createdAt"`
	UpdatedAt   *time.Time `json:"updatedAt"`
}
