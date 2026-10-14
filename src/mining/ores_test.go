package mining

import "testing"

func TestBaseOreName(t *testing.T) {
	for name, want := range map[string]string{
		"Veldspar":                     "Veldspar",
		"Veldspar III-Grade":           "Veldspar",
		"Dark Ochre IV-Grade":          "Dark Ochre",
		"Compressed Veldspar II-Grade": "Veldspar",
		"Batch Compressed Scordite":    "Scordite",
		"Bitumens II-Grade":            "Bitumens",
		"Blue Ice":                     "Blue Ice",
	} {
		if got := BaseOreName(name); got != want {
			t.Errorf("BaseOreName(%q) = %q, want %q", name, got, want)
		}
	}
}
