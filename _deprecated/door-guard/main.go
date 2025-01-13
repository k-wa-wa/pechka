package main

import (
	"fmt"
	"log/slog"

	"github.com/google/go-cmp/cmp"
	_ "golang.org/x/crypto/x509roots/fallback"
	"tinygo.org/x/bluetooth"

	"k-wa-wa/door-guard/internal/env"
	"k-wa-wa/door-guard/internal/iphone"
	"k-wa-wa/door-guard/internal/slack"
	"k-wa-wa/door-guard/internal/switchbot"
)

var adapter = bluetooth.DefaultAdapter

func main() {
	d := DoorGuard{
		onDoorOpened: OnDoorOpened,
	}
	if err := d.Run(); err != nil {
		panic(err)
	}
}

func OnDoorOpened() error {
	isIphoneNearby, err := iphone.IsIphoneNearby(env.IPHONE_HOSTNAME)
	if err != nil {
		return err
	}

	if !isIphoneNearby {
		slog.Warn("isIphoneNearby: false")
		return slack.PostMessage("<!channel> 玄関のドアが開けられました。iPhoneは近くにありません。")
	} else {
		return slack.PostMessage("玄関のドアが開けられました。iPhoneは近くにあります。")
	}
}

type DoorGuard struct {
	onDoorOpened func() error
}

func (dg *DoorGuard) Run() error {
	if err := adapter.Enable(); err != nil {
		return err
	}

	prevSensorMessage := &switchbot.SensorMessage{}
	err := adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		if device.Address.String() == env.SENSOR_MAC {
			for _, s := range device.ServiceData() {
				sensorMessage := switchbot.SensorMessageFromBytes(s.Data)

				if cmp.Diff(prevSensorMessage, sensorMessage, cmp.AllowUnexported(switchbot.SensorMessage{})) != "" {
					slog.Info(fmt.Sprintf("%+v", sensorMessage))

					// ドアが開いた時
					if prevSensorMessage.HalState != switchbot.DoorOpen && sensorMessage.HalState == switchbot.DoorOpen {
						go func() {
							if err := dg.onDoorOpened(); err != nil {
								slog.Error(fmt.Sprint(err))
							}
						}()
					}

					prevSensorMessage = sensorMessage
				}

			}
		}
	})
	if err != nil {
		return err
	}

	return nil
}
