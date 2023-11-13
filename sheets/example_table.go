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

func SetupTablesDB() {
	Open()
	InitSheetsTables()
	teardownTablesDB()
	conn.MustExec(
		`CREATE TABLE IF NOT EXISTS db_interface_test.customers (
    		id SERIAL PRIMARY KEY
			, name VARCHAR(255)
		)`)
	conn.MustExec(
		`CREATE TABLE IF NOT EXISTS db_interface_test.orders (
    		id SERIAL PRIMARY KEY 
			, total DECIMAL
            , status VARCHAR(255)
            , customer_id INT NOT NULL REFERENCES db_interface_test.customers(id)
		)`)
	conn.MustExec(
		`CREATE TABLE IF NOT EXISTS db_interface_test.products (
    		id SERIAL PRIMARY KEY 
			, name VARCHAR(255)
            , price DECIMAL
		)`)
	conn.MustExec(
		`CREATE TABLE IF NOT EXISTS db_interface_test.order_products (
			order_id INT NOT NULL REFERENCES db_interface_test.orders(id)
			, product_id INT NOT NULL REFERENCES db_interface_test.products(id)
		)`)
	loadTables()
}

func LoadExampleData() {
	conn.MustExec(
		`INSERT INTO db_interface_test.customers (name)
			VALUES ('Alice'), ('Bob'), ('Charles'), ('Devon'), ('Erin'), ('Finnegan')
		`)
	conn.MustExec(
		`INSERT INTO db_interface_test.orders (customer_id, total, status)
			VALUES (1, 123.45, 'unfilled'),
				(2, 10, 'shipped'),
				(2, 15, 'delivered'),
				(3, 2000, 'unfilled'),
				(4, 180.20, 'shipped'),
				(4, 41.55, 'shipped'),
				(6, 0.99, 'delivered')
		`)
	conn.MustExec(
		`INSERT INTO db_interface_test.products (name, price)
			VALUES ('ACME Widget', 123.45),
				('Steel Bolt', 10),
				('Chinese Five Spice', 5),
				('Candles', 2000),
				('U-Turn Sign', 180.20),
				('Pillow', 30.56),
				('Digital Download', 0.99)
		`)
	conn.MustExec(
		`INSERT INTO db_interface_test.order_products (order_id, product_id)
			VALUES (1, 1),
				(2, 2),
				(3, 2),
				(3, 3),
				(4, 4),
				(5, 5),
				(6, 2),
				(6, 6),
				(6, 7),
				(7, 7)
		`)
}

func teardownTablesDB() {
	conn.MustExec("DELETE FROM db_interface.sheets WHERE schemaname='db_interface_test'")
	conn.MustExec("DROP TABLE IF EXISTS db_interface_test.customers CASCADE")
	conn.MustExec("DROP TABLE IF EXISTS db_interface_test.orders CASCADE")
	conn.MustExec("DROP TABLE IF EXISTS db_interface_test.products CASCADE")
	conn.MustExec("DROP TABLE IF EXISTS db_interface_test.order_products CASCADE")
}
