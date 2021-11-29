package textselector_test

import (
	"testing"

	textselector "github.com/ipld/go-ipld-selector-text-lite"
)

func TestBasic(t *testing.T) {

	{
		valid := textselector.Expression("/a/42/b/c/")

		ss, err := textselector.SelectorSpecFromPath(valid, false, nil)
		if err != nil {
			t.Fatalf("Expected no error with valid path '%s'", valid)
		}
		if ss == nil {
			t.Fatalf("Expected a selector from valid path '%s'", valid)
		}
	}

	for _, invalid := range []textselector.Expression{
		"/",
		"//",
		"//x",
		"x//",
		";",
	} {
		ss, err := textselector.SelectorSpecFromPath(invalid, false, nil)

		if ss != nil {
			t.Fatalf("Expected nil selector with invalid path '%s'", invalid)
		}

		if err == nil {
			t.Fatalf("Expected error with invalid path '%s'", invalid)
		}
	}
}
