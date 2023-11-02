package sheets

import (
	"testing"
)

func TestDefaultColumnName(t *testing.T) {
	names := map[int]string{
		0:         "A",
		1:         "B",
		26:        "AA",
		27:        "AB",
		52:        "BA",
		26 * 26:   "ZA",
		27 * 26:   "AAA",
		27*26 + 1: "AAB",
	}
	for i, expectedName := range names {
		name := defaultColumnName(i)
		if name != expectedName {
			t.Errorf("%s != %s", name, expectedName)
		}
	}
}
