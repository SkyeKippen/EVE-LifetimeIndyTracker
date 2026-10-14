// Package mining adds up what characters have mined.
package mining

import (
	"regexp"
	"strings"
)

// OreTotal is how much of one base ore has been mined, with its graded
// variants (e.g. "Veldspar III-Grade") counted as the base ore.
type OreTotal struct {
	BaseOre  string
	Quantity int64
	Variants []string // every type mined that counts as this ore, base included
}

// gradeSuffix matches the grade EVE puts after a variant ore's name.
var gradeSuffix = regexp.MustCompile(`\s+[IVX]+-Grade$`)

// BaseOreName turns an ore's name into its base ore, e.g.
// "Compressed Veldspar III-Grade" into "Veldspar".
func BaseOreName(name string) string {
	name = strings.TrimPrefix(name, "Batch Compressed ")
	name = strings.TrimPrefix(name, "Compressed ")
	return gradeSuffix.ReplaceAllString(name, "")
}
