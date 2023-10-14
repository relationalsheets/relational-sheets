package sheets

import (
	"golang.org/x/exp/maps"
	"testing"
)

func setupTablesDB() func() {
	Open()
	conn.MustExec("CREATE SCHEMA IF NOT EXISTS db_interface_test")
	conn.MustExec(
		`CREATE TABLE IF NOT EXISTS db_interface_test.t1 (
    		id SERIAL PRIMARY KEY
			, bar VARCHAR(255)
			, baz INT
		)`)
	conn.MustExec(
		`CREATE TABLE IF NOT EXISTS db_interface_test.t2 (
    		id SERIAL PRIMARY KEY 
			, bar VARCHAR(255)
            , t1_id INT REFERENCES db_interface_test.t1(id)
		)`)
	loadTables()

	return func() {
		conn.MustExec("DROP TABLE IF EXISTS db_interface_test.t1 CASCADE")
		conn.MustExec("DROP TABLE IF EXISTS db_interface_test.t2 CASCADE")
		Check(conn.Close())
	}
}

func TestInsertRow(t *testing.T) {
	teardown := setupTablesDB()
	defer teardown()

	tableName := "db_interface_test.t1"
	sheet := Sheet{Table: TableMap[tableName]}
	tx := Begin()
	row, err := sheet.InsertRow(tx, tableName, map[string]string{"bar": "test"}, []string{"id"})
	if err != nil {
		t.Fatal(err)
	}
	err = tx.Commit()
	if err != nil {
		t.Fatal(err)
	}
	id, _ := row[0].(string)
	if id != "1" {
		t.Fatalf("Unexpected ID returned: %v", row[0])
	}
}

func TestInsertRows(t *testing.T) {
	teardown := setupTablesDB()
	defer teardown()

	tableName := "db_interface_test.t1"
	tableName2 := "db_interface_test.t2"
	table := TableMap[tableName]
	table.loadConstraints()
	sheet := Sheet{Table: table}
	sheet.JoinOids = maps.Keys(table.Fkeys)

	err := sheet.InsertMultipleRows(map[string]map[string]string{
		tableName:  {"bar": "test"},
		tableName2: {"bar": "test2"},
	})
	if err != nil {
		t.Fatal(err)
	}
}
