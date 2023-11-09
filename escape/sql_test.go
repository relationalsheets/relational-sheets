package escape

import (
	"testing"
)

func expectSuccess(t *testing.T, actual, expected string, err error) {
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if actual != expected {
		t.Errorf("%s != %s", actual, expected)
	}
}

func TestEscapeIdentifier(t *testing.T) {
	quoted, err := escapeIdentifier("foo")
	expectSuccess(t, quoted, "\"foo\"", err)

	quoted, err = escapeIdentifier("foo.bar.baz")
	expectSuccess(t, quoted, "\"foo\".\"bar\".\"baz\"", err)

	evil := "\"foo\"; DELETE FROM users;--"
	quoted, err = escapeIdentifier(evil)
	if err == nil {
		t.Errorf("%s should have errored, returned: %s", evil, quoted)
	}
}

func TestIsConstant(t *testing.T) {
	good := "100"
	if !isConstant(good) {
		t.Error(good)
	}

	good = "'foo'"
	if !isConstant(good) {
		t.Error(good)
	}

	evil := "bar; DROP TABLE users;--"
	if isConstant(evil) {
		t.Error(evil)
	}
}

func TestMakeFilterClause(t *testing.T) {
	clause, err := MakeFilterClause("foo", "<1")
	expectSuccess(t, clause.raw, "\"foo\" < 1", err)

	clause, err = MakeFilterClause("1", "< foo")
	expectSuccess(t, clause.raw, "1 < \"foo\"", err)

	clause, err = MakeFilterClause("foo", "= bar")
	expectSuccess(t, clause.raw, "\"foo\" = \"bar\"", err)

	clause, err = MakeFilterClause("foo", "LIKE 'baz%'")
	expectSuccess(t, clause.raw, "\"foo\" LIKE 'baz%'", err)

	evil := "= bar; DELETE FROM users;--"
	clause, err = MakeFilterClause("foo", evil)
	// Quoting makes this safe, although presumably it's not a real column name
	expectSuccess(t, clause.raw, "\"foo\" = \"bar; DELETE FROM users;--\"", err)

	evil = "= bar\"; DELETE FROM users;--"
	clause, err = MakeFilterClause("foo", evil)
	if err == nil {
		t.Errorf("%s should have errored, returned: %s", evil, clause.raw)
	}
}