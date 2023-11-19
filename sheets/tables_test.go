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
	"testing"
)

func TestSingleTableSheet(t *testing.T) {
	SetupTablesDB()
	defer teardownTablesDB()

	tableName := "test.customers"
	sheet := Sheet{}
	sheet.SetTable(tableName)
	sheet.LoadRows(100, 0)

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

	customers := TableMap["test.customers"]
	customers.loadConstraints(nil)
	orders := TableMap["test.orders"]
	orders.loadConstraints(nil)
	products := TableMap["test.products"]
	products.loadConstraints(nil)
	order_products := TableMap["test.order_products"]
	order_products.loadConstraints(nil)
	sheet := Sheet{}
	sheet.SetTable(customers.FullName())
	sheet.JoinOids = make([]int64, 0, 100)
	// Join orders
	for oid, fkey := range orders.Fkeys {
		if fkey.TargetTableName == customers.FullName() {
			err := sheet.SetJoin(0, oid)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	// Join order_products
	for oid, fkey := range order_products.Fkeys {
		if fkey.TargetTableName == orders.FullName() {
			err := sheet.SetJoin(1, oid)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	// Join products
	for oid, fkey := range order_products.Fkeys {
		if fkey.TargetTableName == products.FullName() {
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

	customers := TableMap["test.customers"]
	customers.loadConstraints(nil)
	orders := TableMap["test.orders"]
	orders.loadConstraints(nil)
	products := TableMap["test.products"]
	products.loadConstraints(nil)
	order_products := TableMap["test.order_products"]
	order_products.loadConstraints(nil)
	sheet := Sheet{}
	sheet.SetTable(orders.FullName())
	sheet.JoinOids = make([]int64, 0, 100)
	// Join customers
	for oid, fkey := range orders.Fkeys {
		if fkey.TargetTableName == customers.FullName() {
			err := sheet.SetJoin(0, oid)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	// Join order_products
	for oid, fkey := range orders.Fkeys {
		if fkey.SourceTableName == order_products.FullName() {
			err := sheet.SetJoin(1, oid)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	// Join products
	for oid, fkey := range order_products.Fkeys {
		if fkey.TargetTableName == products.FullName() {
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

