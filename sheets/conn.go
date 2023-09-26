package sheets

import (
	"github.com/jmoiron/sqlx"
	"os"
)

var conn *sqlx.DB

func Open() *sqlx.DB {
	var err error
	conn, err = sqlx.Open("pgx", os.Getenv("DATABASE_URL"))
	Check(err)
	return conn
}
