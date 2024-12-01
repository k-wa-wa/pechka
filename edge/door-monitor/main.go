package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"tinygo.org/x/bluetooth"

	"k-wa-wa/door-monitor/internal/domain"
	"k-wa-wa/door-monitor/internal/env"
	"k-wa-wa/door-monitor/internal/persistence"
)

var adapter = bluetooth.DefaultAdapter

type DoorMonitor struct {
	onStateChanged func(newSensorMessage *domain.ContactSensorMessage) error
}

func main() {
	dm := DoorMonitor{
		onStateChanged: func(sensorMessage *domain.ContactSensorMessage) error {
			// 書き込み頻度が高くないため、毎回コネクションを張る実装とする
			conn, err := pgx.Connect(context.Background(), env.DATABASE_URL)
			if err != nil {
				return err
			}
			defer conn.Close(context.Background())

			db := persistence.NewDB(conn)
			sensorMessageDataRepo := persistence.NewContactSensorDataRepo(db)
			if err := sensorMessageDataRepo.Insert(sensorMessage); err != nil {
				return err
			}

			return nil
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
					go func() {
						if err := dm.onStateChanged(sensorMessage); err != nil {
							slog.Error(fmt.Sprint(err))
						}
					}()
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
