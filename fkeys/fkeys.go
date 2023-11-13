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
package fkeys

import "strings"

type ForeignKey struct {
	SourceTableName string
	TargetTableName string
	SourceColNames  []string
	TargetColNames  []string
}

func (fkey ForeignKey) ToString() string {
	return strings.Join([]string{
		fkey.SourceTableName,
		".",
		strings.Join(fkey.SourceColNames, ","),
		"->",
		fkey.TargetTableName,
		".",
		strings.Join(fkey.TargetColNames, ","),
	}, "")
}
