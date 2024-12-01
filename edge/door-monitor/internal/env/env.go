package env

import "os"

var (
	SENSOR_MAC   = os.Getenv("SENSOR_MAC")
	DATABASE_URL = os.Getenv("DATABASE_URL")
)
