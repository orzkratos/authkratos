package authkratosroutes

import (
	"testing"

	"github.com/yyle88/neatjson/neatjsons"
)

func TestNewPathsMap(t *testing.T) {
	res := NewPathsMap([]Path{
		"a/b/c",
		"x/y/z",
	})
	t.Log(neatjsons.S(res))
}
