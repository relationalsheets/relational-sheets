package sheets

import (
	"testing"
)

func TestSingleTableSheet(t *testing.T) {
	SetupTablesDB()
	defer teardownTablesDB()

	tableName := "db_interface_test.customers"
	sheet := Sheet{}
	sheet.SetTable(tableName)

	// Insert
	tx := Begin()
	row, err := sheet.InsertRow(tx, tableName, map[string]string{"name": "test"}, []string{"id"})
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

	// Update
	err = TableMap[tableName].updateRow(map[string]string{
		"name": "test2",
	}, map[string]string{
		"id": "1",
	})
	if err != nil {
		t.Error(err)
	}

	// Hide
	pref := Pref{TableName: tableName, ColumnName: "name", Hide: true}
	sheet.SavePref(pref)
	sheet.LoadRows(100, 0)
	if len(sheet.Cells[0]) != 1 {
		t.Fatalf("Unexpected number of columns: %v", sheet.Cells[0])
	}
	if len(sheet.Cells[0][0]) != 1 {
		t.Fatalf("Unexpected number of sheet.Cells: %v", sheet.Cells[0][0])
	}
	if sheet.Cells[0][0][0].Value != "1" {
		t.Fatalf("Unexpected id returned: %v", sheet.Cells[0][0][0])
	}

	// Add column with static data
	if len(sheet.ExtraCols) != 0 {
		t.Fatalf("Extra column already exists")
	}
	sheet.AddColumn("")
	sheet.SetCell(0, 0, "abc")
	sheet.loadExtraCols()
	if len(sheet.ExtraCols) != 1 {
		t.Fatalf("Unexpected number of extra columns: %v", sheet.ExtraCols)
	}
	cellValue := sheet.ExtraCols[0].Cells[0].Value
	if cellValue != "abc" {
		t.Fatalf("Unexpected cell value: %s", cellValue)
	}
}

func TestMultiTableSheet(t *testing.T) {
	SetupTablesDB()
	defer teardownTablesDB()

	customers := TableMap["db_interface_test.customers"]
	customers.loadConstraints(nil)
	orders := TableMap["db_interface_test.orders"]
	orders.loadConstraints(nil)
	products := TableMap["db_interface_test.products"]
	products.loadConstraints(nil)
	order_products := TableMap["db_interface_test.order_products"]
	order_products.loadConstraints(nil)
	sheet := Sheet{}
	sheet.SetTable(customers.FullName())
	sheet.JoinOids = make([]int64, 0, 100)
	// Join orders
	for oid, fkey := range customers.Fkeys {
		if fkey.targetTableName == customers.FullName() {
			err := sheet.SetJoin(0, oid)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	// Join order_products
	for oid, fkey := range order_products.Fkeys {
		if fkey.targetTableName == orders.FullName() {
			err := sheet.SetJoin(1, oid)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	// Join products
	for oid, fkey := range order_products.Fkeys {
		if fkey.targetTableName == products.FullName() {
			err := sheet.SetJoin(2, oid)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// Test single insertion and set up data for next insertion
	values := map[string]map[string]string{
		products.FullName(): {"name": "test"},
	}
	err := sheet.InsertMultipleRows(values, map[string]map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	productId := values["product"]["id"]

	// Test insertion with an existing product
	values = map[string]map[string]string{
		customers.FullName():      {"name": "bob"},
		orders.FullName():         {"total": "123.45", "status": "unfilled"},
		order_products.FullName(): {"product_id": "1"},
	}
	err = sheet.InsertMultipleRows(values, map[string]map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if values["order_products"]["product_id"] != productId {
		t.Error("Order not linked to inserted product")
	}

	// Test insertion with a new product
	err = sheet.InsertMultipleRows(map[string]map[string]string{
		customers.FullName(): {"name": "bob"},
		orders.FullName():    {"total": "123.45", "status": "unfilled"},
		products.FullName():  {"name": "test"},
	}, map[string]map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	// Test insertion with an existing customer not directly referenced in values
	err = sheet.InsertMultipleRows(map[string]map[string]string{
		orders.FullName(): {"total": "123.45", "status": "unfilled"},
	}, map[string]map[string]string{
		customers.FullName(): {"id": "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAlternateJoinOrder(t *testing.T) {
	SetupTablesDB()
	defer teardownTablesDB()

	customers := TableMap["db_interface_test.customers"]
	customers.loadConstraints(nil)
	orders := TableMap["db_interface_test.orders"]
	orders.loadConstraints(nil)
	products := TableMap["db_interface_test.products"]
	products.loadConstraints(nil)
	order_products := TableMap["db_interface_test.order_products"]
	order_products.loadConstraints(nil)
	sheet := Sheet{}
	sheet.SetTable(orders.FullName())
	sheet.JoinOids = make([]int64, 0, 100)
	// Join customers
	for oid, fkey := range orders.Fkeys {
		if fkey.targetTableName == customers.FullName() {
			err := sheet.SetJoin(0, oid)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	// Join order_products
	for oid, fkey := range orders.Fkeys {
		if fkey.sourceTableName == order_products.FullName() {
			err := sheet.SetJoin(1, oid)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	// Join products
	for oid, fkey := range order_products.Fkeys {
		if fkey.targetTableName == products.FullName() {
			err := sheet.SetJoin(2, oid)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// Test single insertion and set up data for next insertion
	values := map[string]map[string]string{
		products.FullName(): {"name": "test"},
	}
	err := sheet.InsertMultipleRows(values, map[string]map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	productId := values["product"]["id"]

	// Test insertion with an existing product
	values = map[string]map[string]string{
		customers.FullName():      {"name": "bob"},
		orders.FullName():         {"total": "123.45", "status": "unfilled"},
		order_products.FullName(): {"product_id": "1"},
	}
	err = sheet.InsertMultipleRows(values, map[string]map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if values["order_products"]["product_id"] != productId {
		t.Error("Order not linked to inserted product")
	}

	// Test insertion with a new product
	err = sheet.InsertMultipleRows(map[string]map[string]string{
		customers.FullName(): {"name": "bob"},
		orders.FullName():    {"total": "123.45", "status": "unfilled"},
		products.FullName():  {"name": "test"},
	}, map[string]map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	// Test insertion with an existing customer not directly referenced in values
	err = sheet.InsertMultipleRows(map[string]map[string]string{
		orders.FullName(): {"total": "123.45", "status": "unfilled"},
	}, map[string]map[string]string{
		customers.FullName(): {"id": "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

