package sde

import (
	"strings"
	"testing"
)

func TestParseTypeMaterials(t *testing.T) {
	got, err := ParseTypeMaterials(strings.NewReader(
		`{"_key":1230,"materials":[{"materialTypeID":34,"quantity":400}]}` + "\n" +
			`{"_key":18,"materials":[{"materialTypeID":34,"quantity":175},{"materialTypeID":35,"quantity":70}]}` + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1230][0] != (Material{34, 400}) || got[18][1] != (Material{35, 70}) {
		t.Errorf("got %v", got)
	}
}
