package sheets

import (
	"golang.org/x/exp/maps"
	"testing"
)

func TestSingleTableSheet(t *testing.T) {
	SetupTablesDB()
	defer teardownTablesDB()

	tableName := "db_interface_test.customers"
	sheet := Sheet{Table: TableMap[tableName]}

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
	sheet := Sheet{Table: customers}
	// Join orders
	sheet.JoinOids = maps.Keys(customers.Fkeys)
	// Join order_products
	for oid, fkey := range orders.Fkeys {
		if fkey.sourceTableName == order_products.FullName() {
			sheet.JoinOids = append(sheet.JoinOids, oid)
		}
	}
	// Join products
	for oid, fkey := range order_products.Fkeys {
		if fkey.targetTableName == products.FullName() {
			sheet.JoinOids = append(sheet.JoinOids, oid)
		}
	}

	sheet.loadJoins()

	// Test single insertion and set up data for next insertion
	values := map[string]map[string]string{
		products.FullName(): {"name": "test"},
	}
	err := sheet.InsertMultipleRows(values)
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
	err = sheet.InsertMultipleRows(values)
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
	})
	if err != nil {
		t.Fatal(err)
	} 
}
