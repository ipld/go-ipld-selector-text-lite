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

	basicnode "github.com/ipld/go-ipld-prime/node/basic"
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
func SelectorSpecFromPath(path Expression, optionalSubselectorAtTarget builder.SelectorSpec) (builder.SelectorSpec, error) {

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
	}

	return ss, nil
}
