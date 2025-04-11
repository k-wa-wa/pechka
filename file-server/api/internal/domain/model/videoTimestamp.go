package model

type VideoTimestampEntity struct {
	TimestampId string `json:"timestampId"`
	VideoId     string `json:"videoId"`
	Timestamp   string `json:"timestamp"`
	Description string `json:"description"`
}
