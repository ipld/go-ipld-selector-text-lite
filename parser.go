/*

Package textselector provides basic utilities for creation of IPLD selector
objects from a flat textual representation. For further info see
https://github.com/ipld/specs/blob/master/selectors/selectors.md
*/
package textselector

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/ipld/go-ipld-prime/traversal/selector"
	"github.com/ipld/go-ipld-prime/traversal/selector/builder"
)

const (
	PathValidCharset = `[- _0-9a-zA-Z\/\.]`
)

var invalidChar = regexp.MustCompile(`[^` + PathValidCharset[1:len(PathValidCharset)-1] + `]`)
var onlyDigits = regexp.MustCompile(`^[0-9]+$`)

/*
SelectorFromPath transforms a textual path specification in the form x/y/10/z
into a go-ipld-prime selector object. This is a short-term stop-gap on the road
to a more versatile text-based selector specification. Therefore the accepted
syntax is relatively inflexible, and restricted to the members of
PathValidCharset. The parsing rules are:

	- The character `/` is a path segment separator
	- An empty segment ( `...//...` ) and the unix-like `.` and `..` are illegal
	- A segment composed entirely of digits `[0-9]+` is treated as an array index
	- Any other valid segment is treated as a hash key within a map

*/
func SelectorFromPath(path string) (selector.Selector, error) {

	if path == "/" {
		return nil, fmt.Errorf("a standalone '/' is not a valid path")
	} else if m := invalidChar.FindStringIndex(path); m != nil {
		return nil, fmt.Errorf("path string contains invalid character at offset %d", m[0])
	}

	ssb := builder.NewSelectorSpecBuilder(basicnode.Style.Any)

	// start from a matcher and walk backwards wrapping it recursively
	ss := ssb.Matcher()

	segments := strings.Split(path, "/")

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

		if onlyDigits.MatchString(segments[i]) {
			if segments[i][0] == '0' && len(segments[i]) > 1 {
				return nil, fmt.Errorf("invalid segment '%s' at position %d", segments[i], i)
			}

			idx, err := strconv.ParseInt(segments[i], 10, 31)
			if err != nil {
				return nil, fmt.Errorf("invalid index '%s' at position %d: %s", segments[i], i, err)
			}

			ss = ssb.ExploreIndex(
				int(idx),
				ss,
			)
		} else {
			ss = ssb.ExploreFields(func(efsb builder.ExploreFieldsSpecBuilder) {
				efsb.Insert(segments[i], ss)
			})
		}
	}

	return selector.ParseSelector(ss.Node())
}
