package switchbot

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSensorMessageFromBytes(t *testing.T) {
	testcases := []struct {
		data     []byte
		expected *SensorMessage
	}{
		{
			data: []byte{
				0b0,
				0b0,
				0b0,
				0b00000010,
			},
			expected: &SensorMessage{
				HalState:   "door-open",
				LightLevel: "dark",
			},
		},
		{
			data: []byte{
				0b0,
				0b0,
				0b0,
				0b00000100,
			},
			expected: &SensorMessage{
				HalState:   "timeout-not-close",
				LightLevel: "dark",
			},
		},
	}

	for _, testcase := range testcases {
		sensorMessage := SensorMessageFromBytes(testcase.data)
		diff := cmp.Diff(testcase.expected, sensorMessage, cmp.AllowUnexported(SensorMessage{}))
		if diff != "" {
			t.Error(diff)
		}
	}
}
