package switchbot

type SensorMessage struct {
	LightLevel lightLevel
	HalState   halState
}

type lightLevel string

const Dark = lightLevel("dark")
const Light = lightLevel("light")

func lightLevelFromInt(i int) lightLevel {
	switch i {
	case 0:
		return Dark
	case 1:
		return Light
	}
	return ""
}

type halState string

const DoorClose = halState("door-close")
const DoorOpen = halState("door-open")
const TimeoutNotClose = halState("timeout-not-close")

func halStateFromInt(i int) halState {
	switch i {
	case 0:
		return DoorClose
	case 1:
		return DoorOpen
	case 2:
		return TimeoutNotClose
	}
	return ""
}

func SensorMessageFromBytes(data []byte) *SensorMessage {
	return &SensorMessage{
		LightLevel: lightLevelFromInt(int(data[3] & 1)),
		HalState:   halStateFromInt(int((data[3] >> 1) & 0b11)),
	}
}
