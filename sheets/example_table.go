package sheets

func SetupTablesDB() {
	Open()
	conn.MustExec("CREATE SCHEMA IF NOT EXISTS db_interface_test")
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

func teardownTablesDB() {
	conn.MustExec("DROP TABLE IF EXISTS db_interface_test.customers CASCADE")
	conn.MustExec("DROP TABLE IF EXISTS db_interface_test.orders CASCADE")
	conn.MustExec("DROP TABLE IF EXISTS db_interface_test.products CASCADE")
	conn.MustExec("DROP TABLE IF EXISTS db_interface_test.order_products CASCADE")
	Check(conn.Close())
}
