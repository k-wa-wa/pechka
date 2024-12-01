package persistence

import "context"

type sensorQueueRepo struct {
	db *DB
}

func NewSensorQueueRepo(db *DB) *sensorQueueRepo {
	return &sensorQueueRepo{db}
}

func (sqr *sensorQueueRepo) Insert(message string) error {
	conn, err := sqr.db.pool.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(
		context.Background(),
		`INSERT INTO sensor_queue (message) VALUES (%s)`,
		message,
	); err != nil {
		return err
	}

	return nil
}
