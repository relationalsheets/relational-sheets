package sheets

import (
	"log"
	"slices"
)

func initExtraColsTables() {
	conn.MustExec(`
		CREATE TABLE IF NOT EXISTS db_interface.sheetcols (
			id SERIAL PRIMARY KEY
			, sheet_id INT NOT NULL
			, i INTEGER NOT NULL
			, colname VARCHAR(255) NOT NULL
			, UNIQUE (sheet_id, i)
			, CONSTRAINT fk_sheets
				FOREIGN KEY (sheet_id)
					REFERENCES db_interface.sheets(id) ON DELETE CASCADE
		)`)
	log.Println("SheetCols table exists")

	conn.MustExec(`
		CREATE TABLE IF NOT EXISTS db_interface.sheetcells (
			id SERIAL PRIMARY KEY
			, sheetcol_id INT NOT NULL
			, j INTEGER NOT NULL
			, formula VARCHAR(255) NOT NULL
			, UNIQUE (sheetcol_id, j)
			, CONSTRAINT fk_sheetcol
				FOREIGN KEY (sheetcol_id)
					REFERENCES db_interface.sheetcols(id) ON DELETE CASCADE
		)`)
	log.Println("SheetCells table exists")
}

func (s *Sheet) loadCells() {
	for i, col := range s.ExtraCols {
		col.Cells = make([]SheetCell, 100)
		s.ExtraCols[i] = col
	}

	rows, err := conn.Query(`
		SELECT i, j, Formula
		FROM db_interface.sheetcells
			JOIN db_interface.sheetcols
			ON sheetcol_id = db_interface.sheetcols.id
		WHERE sheet_id = $1
		ORDER BY i, j`,
		s.Id)
	Check(err)

	var formula string
	var i, j int
	for rows.Next() {
		err = rows.Scan(&i, &j, &formula)
		Check(err)
		s.ExtraCols[i].Cells[j], err = s.EvalFormula(formula)
		if err != nil {
			log.Printf("Error loading cell %d,%d (%s): %s", i, j, formula, err)
		}
	}

	log.Println("Loaded custom column cells")
}

func (s *Sheet) loadExtraCols() {
	s.ExtraCols = make([]SheetColumn, 0, 20)
	err := conn.Select(&s.ExtraCols, `
		SELECT id
			, colname AS "name"
		FROM db_interface.sheetcols
		WHERE sheet_id = $1
		ORDER BY i`,
		s.Id)
	Check(err)
	log.Printf("Loaded %d custom columns", len(s.ExtraCols))

	s.loadCells()
}

func (s *Sheet) saveCol(i int) {
	log.Printf("Saving extra column %d", i)
	col := s.ExtraCols[i]
	row := conn.QueryRow(`
		INSERT INTO db_interface.sheetcols (
			sheet_id
			, i
			, colname
		) VALUES (
			$1, $2, $3
		) ON CONFLICT (sheet_id, i) DO
		UPDATE SET colname = $3
		RETURNING id`,
		s.Id,
		i,
		col.Name)
	err := row.Scan(&col.Id)
	Check(err)
	s.ExtraCols[i] = col
}

func (s *Sheet) SetCell(i, j int, formula string) SheetCell {
	column := s.ExtraCols[i]
	cell, err := s.EvalFormula(formula)
	Check(err)
	column.Cells[j] = cell
	log.Printf("Saving cell (%d,%d) into column id=%d", i, j, s.ExtraCols[i].Id)
	conn.MustExec(`
		INSERT INTO db_interface.sheetcells (
		    sheetcol_id
		    , j
		    , formula
		) VALUES ($1, $2, $3)
		ON CONFLICT (sheetcol_id, j) DO
		UPDATE SET formula = $3`,
		s.ExtraCols[i].Id,
		j,
		formula)
	return cell
}

func defaultColumnName(i int) string {
	name := ""
	n := len(defaultColNameChars)
	i += 1
	for i > 0 {
		name = defaultColNameChars[(i-1)%n:(i-1)%n+1] + name
		i = (i - (i-1)%n) / n
	}
	return name
}

func (s *Sheet) AddColumn(name string) {
	if name == "" {
		name = defaultColumnName(len(s.ExtraCols))
	}
	log.Printf("Adding column %s to sheet %d", name, s.Id)

	s.ExtraCols = append(s.ExtraCols, SheetColumn{Name: name, Cells: make([]SheetCell, 100)})
	s.saveCol(len(s.ExtraCols) - 1)
	SheetMap[s.Id] = *s
}

func (s *Sheet) RenameCol(i int, name string) {
	col := s.ExtraCols[i]
	col.Name = name
	s.ExtraCols[i] = col
	s.saveCol(i)
}

func (s *Sheet) DeleteColumn(i int) {
	conn.MustExec(
		"DELETE FROM db_interface.sheetcols WHERE id = $1",
		s.ExtraCols[i].Id)
	conn.MustExec(`
		UPDATE db_interface.sheetcols
		SET i = i + 1
		WHERE sheet_id = $1 and i > $2`,
		s.Id,
		i)
	s.ExtraCols = slices.Delete(s.ExtraCols, i, i+1)
	SheetMap[s.Id] = *s
}
