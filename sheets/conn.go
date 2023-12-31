// This file is part of Relational Sheets.
//
// Relational Sheets is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free Software Foundation,
// either version 3 of the License, or (at your option) any later version.
//
// Relational Sheets is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
// without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
// See the GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU General Public License along with Relational Sheets.
// If not, see https://www.gnu.org/licenses/agpl-3.0.html
package sheets

import (
	"github.com/jmoiron/sqlx"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var conn *sqlx.DB

func Open() *sqlx.DB {
	var err error
	conn, err = sqlx.Open("pgx", os.Getenv("DATABASE_URL"))
	Check(err)
	return conn
}

func Begin() *sqlx.Tx {
	return conn.MustBegin()
}

func Commit(tx *sqlx.Tx) {
	Check(tx.Commit())
}
