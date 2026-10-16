package production

import "testing"

func names(items []Item) string {
	out := ""
	for _, item := range items {
		out += item.Name + ","
	}
	return out
}

func TestSortItems(t *testing.T) {
	items := []Item{
		{Name: "Nitrogen Fuel Block", Category: "Material", Group: "Fuel Block", Quantity: 4000, Value: 80e6},
		{Name: "Light Missile Launcher II", Category: "Module", Group: "Missile Launcher Light", Quantity: 90, Value: 90e6},
		{Name: "Scourge Light Missile", Category: "Charge", Group: "Light Missile", Quantity: 4000, Value: 1e6},
		{Name: "Heron", Category: "Ship", Group: "Frigate", Quantity: 2, Value: 1e6},
	}
	for _, test := range []struct {
		sort Sort
		want string
	}{
		{DefaultSort, "Nitrogen Fuel Block,Scourge Light Missile,Light Missile Launcher II,Heron,"},
		{Sort{SortQuantity, false}, "Heron,Light Missile Launcher II,Nitrogen Fuel Block,Scourge Light Missile,"},
		{Sort{SortName, false}, "Heron,Light Missile Launcher II,Nitrogen Fuel Block,Scourge Light Missile,"},
		{Sort{SortType, false}, "Scourge Light Missile,Nitrogen Fuel Block,Light Missile Launcher II,Heron,"},
		{Sort{SortValue, true}, "Light Missile Launcher II,Nitrogen Fuel Block,Heron,Scourge Light Missile,"},
	} {
		SortItems(items, test.sort)
		if got := names(items); got != test.want {
			t.Errorf("%+v: got %s, want %s", test.sort, got, test.want)
		}
	}
}

func TestParseSortAndToggle(t *testing.T) {
	if got := ParseSort("", ""); got != DefaultSort {
		t.Errorf("default = %+v", got)
	}
	if got := ParseSort("name", ""); got != (Sort{SortName, false}) {
		t.Errorf("name = %+v", got)
	}
	if got := ParseSort("value", "asc"); got != (Sort{SortValue, false}) {
		t.Errorf("value asc = %+v", got)
	}
	if got := DefaultSort.Toggle(SortQuantity); got != (Sort{SortQuantity, false}) {
		t.Errorf("toggle quantity = %+v", got)
	}
	if got := DefaultSort.Toggle(SortType); got != (Sort{SortType, false}) {
		t.Errorf("toggle to type = %+v", got)
	}
}
