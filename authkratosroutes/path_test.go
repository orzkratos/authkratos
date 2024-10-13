package authkratosroutes

import (
	"testing"

	"github.com/yyle88/neatjson/neatjsons"
)

func TestMain(m *testing.M) {
	m.Run()
}

func TestNewPathsBooMap(t *testing.T) {
	res := NewPathsBooMap([]Path{
		"a/b/c",
		"x/y/z",
	})
	t.Log(neatjsons.S(res))
}
