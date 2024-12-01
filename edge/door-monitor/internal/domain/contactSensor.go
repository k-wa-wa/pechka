package domain

import (
	"time"
)

type ContactSensorMessage struct {
	EventTimestamp time.Time
	SensorMac      string
	LightLevel     int // 0: dark, 1: light
	HalState       int // 0: door close, 1: door open, 2: timeout not close

	//event_timestamp timestamp,
	//sensor_mac char(17),
	//light_level int,
	//hal_state int
}

func ContactSensorMessageFromBytes(data []byte, sensorMac string) *ContactSensorMessage {
	return &ContactSensorMessage{
		EventTimestamp: time.Now(),
		SensorMac:      sensorMac,
		LightLevel:     int(data[3] & 1),
		HalState:       int((data[3] >> 1) & 0b11),
	}
}

func ContactSensorMessageHasChanged(prev *ContactSensorMessage, current *ContactSensorMessage) bool {
	return (prev.LightLevel != current.LightLevel) || (prev.HalState != current.HalState)
}
