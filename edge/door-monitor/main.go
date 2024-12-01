package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "golang.org/x/crypto/x509roots/fallback"
	"tinygo.org/x/bluetooth"

	"k-wa-wa/door-monitor/internal/domain"
	"k-wa-wa/door-monitor/internal/env"
	"k-wa-wa/door-monitor/internal/persistence"
	"k-wa-wa/door-monitor/internal/slack"
)

var adapter = bluetooth.DefaultAdapter

type DoorMonitor struct {
	onStateChanged func(newSensorMessage *domain.ContactSensorMessage)
	onDoorOpened   func(sensorMessage *domain.ContactSensorMessage)
}

func main() {
	dbPool, err := pgxpool.New(context.Background(), env.DATABASE_URL)
	if err != nil {
		panic(err)
	}
	defer dbPool.Close()

	dm := DoorMonitor{
		onStateChanged: func(sensorMessage *domain.ContactSensorMessage) {
			db := persistence.NewDB(dbPool)
			sensorMessageDataRepo := persistence.NewContactSensorDataRepo(db)
			if err := sensorMessageDataRepo.Insert(sensorMessage); err != nil {
				slog.Error(err.Error())
			}
		},
		onDoorOpened: func(sensorMessage *domain.ContactSensorMessage) {
			if err := slack.PostMessage("door-opened:" + sensorMessage.SensorMac); err != nil {
				slog.Error(err.Error())
			}

			db := persistence.NewDB(dbPool)
			sensorQueueRepo := persistence.NewSensorQueueRepo(db)
			if err := sensorQueueRepo.Insert("door-opened:" + sensorMessage.SensorMac); err != nil {
				slog.Error(err.Error())
			}
		},
	}
	if err := dm.Run(); err != nil {
		panic(err)
	}
}

func (dm *DoorMonitor) Run() error {
	if err := adapter.Enable(); err != nil {
		return err
	}

	prevSensorMessage := domain.ContactSensorMessage{}
	err := adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		if device.Address.String() == env.SENSOR_MAC {
			for _, s := range device.ServiceData() {
				sensorMessage := domain.ContactSensorMessageFromBytes(s.Data, device.Address.String())

				if domain.ContactSensorMessageHasChanged(&prevSensorMessage, sensorMessage) {
					slog.Info(fmt.Sprintf("%+v", sensorMessage))

					// ドアが開いた時
					if prevSensorMessage.HalState != 1 && sensorMessage.HalState == 1 {
						go dm.onDoorOpened(sensorMessage)
					}

					go dm.onStateChanged(sensorMessage)
					prevSensorMessage = *sensorMessage
				}

			}
		}
	})
	if err != nil {
		return err
	}

	return nil
}
