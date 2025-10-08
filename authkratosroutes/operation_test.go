package authkratosroutes

import (
	"testing"

	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/neatjson/neatjsons"
)

func TestNewOperations(t *testing.T) {
	operations := []Operation{"a/b/c", "x/y/z"}
	set := utils.NewSet(operations)
	t.Log(neatjsons.S(set))
}
