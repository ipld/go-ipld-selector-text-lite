/*
Package textselector provides basic utilities for creation of IPLD selector
objects from a flat textual representation. For further info see
https://github.com/ipld/specs/blob/master/selectors/selectors.md
*/
package textselector

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ipld/go-ipld-prime/codec/dagjson"
	"github.com/ipld/go-ipld-prime/datamodel"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/ipld/go-ipld-prime/traversal/selector"
	"github.com/ipld/go-ipld-prime/traversal/selector/builder"
)

// PathValidCharset is the regular expression fully matching a valid textselector
const PathValidCharset = `[- _0-9a-zA-Z\/\.]`

// Expression is a string-type input to SelectorSpecFromPath
type Expression string

var invalidChar = regexp.MustCompile(`[^` + PathValidCharset[1:len(PathValidCharset)-1] + `]`)

/*
SelectorSpecFromPath transforms a textual path specification in the form x/y/10/z
into a go-ipld-prime selector-spec object. This is a short-term stop-gap on the
road to a more versatile text-based selector description mechanism. Therefore
the accepted syntax is relatively inflexible, and restricted to the members of
PathValidCharset. The parsing rules are:

	- The character `/` is a path segment separator
	- An empty segment ( `...//...` ) and the unix-like `.` and `..` are illegal
	- Any other valid segment is treated as a key within a map, or (if applicable)
	  as an index within an array
*/
func SelectorSpecFromPath(path Expression, matchPath bool, optionalSubselectorAtTarget builder.SelectorSpec) (builder.SelectorSpec, error) {

	if path == "/" {
		return nil, fmt.Errorf("a standalone '/' is not a valid path")
	} else if m := invalidChar.FindStringIndex(string(path)); m != nil {
		return nil, fmt.Errorf("path string contains invalid character at offset %d", m[0])
	}

	ssb := builder.NewSelectorSpecBuilder(basicnode.Prototype.Any)

	ss := optionalSubselectorAtTarget
	// if nothing is given - use an exact matcher
	if ss == nil {
		ss = ssb.Matcher()
	}

	segments := strings.Split(string(path), "/")

	// walk backwards wrapping the original selector recursively
	for i := len(segments) - 1; i >= 0; i-- {
		if segments[i] == "" {
			// allow one leading and one trailing '/' at most
			if i == 0 || i == len(segments)-1 {
				continue
			}
			return nil, fmt.Errorf("invalid empty segment at position %d", i)
		}

		if segments[i] == "." || segments[i] == ".." {
			return nil, fmt.Errorf("unsupported path segment '%s' at position %d", segments[i], i)
		}

		ss = ssb.ExploreFields(func(efsb builder.ExploreFieldsSpecBuilder) {
			efsb.Insert(segments[i], ss)
		})

		if matchPath {
			ss = ssb.ExploreUnion(ssb.Matcher(), ss)
		}
	}

	return ss, nil
}

type selectorSpec struct {
	n datamodel.Node
	s selector.Selector
}

func (ss selectorSpec) Node() datamodel.Node {
	return ss.n
}

func (ss selectorSpec) Selector() (selector.Selector, error) {
	return ss.s, nil
}

/*
SelectorSpecFromJson transforms a string form of a dag-json representation of a
go-ipld-prime selector into a selector-spec object.

The full dag-json form of a selector is not a user-friendly thing to construct,
however it does allow for the representation of the full range of possible
selectors where simple pathing-based selectors won't work.

Note that using this format, the top-level of the input must be wrapped in an
envelope (map) with a single field "selector".

For example, a selector for the path x/y/10/z could be supplied to
SelectorSpecFromJson as:

	{"selector"{"f":{"f>":{"x":{"f":{"f>":{"y":{"f":{"f>":{"10":{"f":{"f>":{"z":{".":{}}}}}}}}}}}}}}}

See https://ipld.io/specs/selectors/ for more information.
*/
func SelectorSpecFromJson(jsonStr string) (builder.SelectorSpec, error) {
	// similar to selectorparse.ParseJSONSelector(input) but we want both the node
	// and the selector, and we want to ensure a `"selector"` top-level field to
	// contain the selector
	nb := basicnode.Prototype.Any.NewBuilder()
	if err := dagjson.Decode(nb, strings.NewReader(jsonStr)); err != nil {
		return nil, err
	}
	n := nb.Build()
	if n.Kind() != datamodel.Kind_Map {
		return nil, fmt.Errorf("expected a map object from dag-json input")
	}
	if n.Length() != 1 {
		return nil, fmt.Errorf("expected a single field map object from dag-json input")
	}
	n, err := n.LookupByString("selector")
	if err != nil {
		return nil, fmt.Errorf(`expected a "selector" field in dag-json input: %v`, err)
	}
	// will error if it's not a proper selector
	sel, err := selector.CompileSelector(n)
	if err != nil {
		return nil, fmt.Errorf("dag-json input is not a valid selector: %v", err)
	}
	return selectorSpec{n, sel}, nil
}
