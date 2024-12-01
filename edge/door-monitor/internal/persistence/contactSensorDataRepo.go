package persistence

import (
	"context"
	"k-wa-wa/door-monitor/internal/domain"
)

type contactSensorDataRepo struct {
	db *DB
}

func NewContactSensorDataRepo(db *DB) *contactSensorDataRepo {
	return &contactSensorDataRepo{db}
}

func (csd *contactSensorDataRepo) Insert(contactSensor *domain.ContactSensorMessage) error {
	conn, err := csd.db.pool.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(
		context.Background(),
		`INSERT INTO contact_sensor_data VALUES (
			$1, $2, $3, $4
		)`,
		contactSensor.EventTimestamp,
		contactSensor.SensorMac,
		contactSensor.LightLevel,
		contactSensor.HalState,
	); err != nil {
		return err
	}

	return nil
}
