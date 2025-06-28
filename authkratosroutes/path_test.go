package authkratosroutes

import (
	"testing"

	"github.com/yyle88/neatjson/neatjsons"
)

func TestNewOperations(t *testing.T) {
	res := NewOperations([]Path{
		"a/b/c",
		"x/y/z",
	})
	t.Log(neatjsons.S(res))
}
