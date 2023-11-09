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
