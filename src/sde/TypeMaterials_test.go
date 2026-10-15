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

func TestParseVolumes(t *testing.T) {
	got, err := ParseVolumes(strings.NewReader(
		`{"_key":1230,"name":{"en":"Veldspar"},"volume":0.1,"portionSize":100}` + "\n" +
			`{"_key":28432,"volume":0.001}` + "\n" + `{"_key":5}` + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1230] != 0.1 || got[28432] != 0.001 {
		t.Errorf("got %v", got)
	}
}
