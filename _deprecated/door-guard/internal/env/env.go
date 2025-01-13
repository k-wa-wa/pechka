package env

import "os"

var (
	SENSOR_MAC        = os.Getenv("SENSOR_MAC")
	IPHONE_HOSTNAME   = os.Getenv("IPHONE_HOSTNAME")
	SLACK_WEBHOOK_URL = os.Getenv("SLACK_WEBHOOK_URL")
)
