// Package production lists and sorts the items a user has made.
package production

import (
	"cmp"
	"slices"
	"strings"
)

// Item is everything a user has made of one item type.
type Item struct {
	Name             string
	Category         string // e.g. Module, Charge, Ship; "" when unknown
	Group            string // e.g. Missile Launcher Light, Fuel Block
	Quantity         int64
	Value            float64 // what the items were worth when delivered
	UnpricedQuantity int64   // items left out of Value for lack of a price
}

// Columns the list can be sorted by.
const (
	SortName     = "name"
	SortType     = "type"
	SortQuantity = "quantity"
	SortValue    = "value"
)

// Sort is a column and direction to sort by.
type Sort struct {
	Column string
	Desc   bool
}

// DefaultSort is most made first.
var DefaultSort = Sort{Column: SortQuantity, Desc: true}

// ParseSort reads the column and direction from the page's query, falling
// back to DefaultSort for an unknown column. dir is "asc" or "desc"; any
// other dir gives the column's natural direction: A to Z for text, largest
// first for numbers.
func ParseSort(column, dir string) Sort {
	switch column {
	case SortName, SortType:
		return Sort{Column: column, Desc: dir == "desc"}
	case SortQuantity, SortValue:
		return Sort{Column: column, Desc: dir != "asc"}
	default:
		return DefaultSort
	}
}

// Toggle is the sort a click on column's heading gives: the other direction
// when the list is already sorted by it, otherwise its natural direction.
func (s Sort) Toggle(column string) Sort {
	if s.Column == column {
		return Sort{Column: column, Desc: !s.Desc}
	}
	return ParseSort(column, "")
}

// Dir is "desc" or "asc", for the page's query.
func (s Sort) Dir() string {
	if s.Desc {
		return "desc"
	}
	return "asc"
}

// SortItems sorts items in place. Ties fall back to the item name, A to Z.
func SortItems(items []Item, s Sort) {
	slices.SortStableFunc(items, func(a, b Item) int {
		var order int
		switch s.Column {
		case SortName:
			order = strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		case SortType:
			order = cmp.Or(strings.Compare(a.Category, b.Category), strings.Compare(a.Group, b.Group))
		case SortQuantity:
			order = cmp.Compare(a.Quantity, b.Quantity)
		case SortValue:
			order = cmp.Compare(a.Value, b.Value)
		}
		if s.Desc {
			order = -order
		}
		return cmp.Or(order, strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)))
	})
}
