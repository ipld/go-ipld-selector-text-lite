package textselector_test

import (
	"testing"

	textselector "github.com/ipld/go-ipld-selector-text-lite"
)

func TestBasic(t *testing.T) {

	{
		valid := textselector.Expression("/a/42/b/c/")

		sel, err := textselector.SelectorFromPath(valid)
		if err != nil {
			t.Fatalf("Expected no error with valid path '%s'", valid)
		}
		if sel == nil {
			t.Fatalf("Expected a selector from valid path '%s'", valid)
		}
	}

	for _, invalid := range []textselector.Expression{
		"/",
		"//",
		"//x",
		"x//",
		"00",
		"01",
		"001",
		";",
	} {
		sel, err := textselector.SelectorFromPath(invalid)

		if sel != nil {
			t.Fatalf("Expected nil selector with invalid path '%s'", invalid)
		}

		if err == nil {
			t.Fatalf("Expected error with invalid path '%s'", invalid)
		}
	}
}
