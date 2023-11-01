package sheets

import "log"

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
			, sheet_id INT NOT NULL
			, i INTEGER NOT NULL
			, j INTEGER NOT NULL
			, formula VARCHAR(255) NOT NULL
			, UNIQUE (sheet_id, i, j)
			, CONSTRAINT fk_sheets
				FOREIGN KEY (sheet_id)
					REFERENCES db_interface.sheets(id) ON DELETE CASCADE
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
		SELECT colname AS "name"
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
	conn.MustExec(`
		INSERT INTO db_interface.sheetcols (
			sheet_id
			, i
			, colname
		) VALUES (
			$1, $2, $3
		) ON CONFLICT (sheet_id, i) DO
		UPDATE SET colname = $3`,
		s.Id,
		i,
		s.ExtraCols[i].Name)
}

func (s *Sheet) SetCell(i, j int, formula string) SheetCell {
	column := s.ExtraCols[i]
	cell, err := s.EvalFormula(formula)
	Check(err)
	column.Cells[j] = cell
	conn.MustExec(`
		INSERT INTO db_interface.sheetcells (
		    sheet_id
		    , i
		    , j
		    , formula
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (sheet_id, i, j) DO
		UPDATE SET formula = $4`,
		s.Id,
		i,
		j,
		formula)
	return cell
}

func (s *Sheet) AddColumn(name string) {
	if name == "" {
		i := len(s.ExtraCols)
		for i >= 0 {
			name += defaultColNameChars[i%len(defaultColNameChars) : i%len(defaultColNameChars)+1]
			i -= len(defaultColNameChars)
		}
	}
	log.Printf("Adding column %s to sheet %d", name, s.Id)

	cells := make([]SheetCell, 100)
	s.ExtraCols = append(s.ExtraCols, SheetColumn{name, cells})
	s.saveCol(len(s.ExtraCols) - 1)
	SheetMap[s.Id] = *s
}

func (s *Sheet) RenameCol(i int, name string) {
	col := s.ExtraCols[i]
	col.Name = name
	s.ExtraCols[i] = col
	s.saveCol(i)
}
