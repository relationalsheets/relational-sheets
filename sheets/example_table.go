package sheets

func SetupTablesDB() {
	Open()
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
	conn.MustExec("DROP TABLE IF EXISTS db_interface_test.customers CASCADE")
	conn.MustExec("DROP TABLE IF EXISTS db_interface_test.orders CASCADE")
	conn.MustExec("DROP TABLE IF EXISTS db_interface_test.products CASCADE")
	conn.MustExec("DROP TABLE IF EXISTS db_interface_test.order_products CASCADE")
}
