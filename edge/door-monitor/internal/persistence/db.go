package persistence

import (
	"github.com/jackc/pgx/v5"
)

type DB struct {
	conn *pgx.Conn
}

func NewDB(conn *pgx.Conn) *DB {
	return &DB{conn}
}
