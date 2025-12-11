package authkratosroutes

import (
	"testing"

	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/neatjson/neatjsons"
)

// TestNewOperations tests creating operation set from slice
// TestNewOperations 测试从切片创建操作集合
func TestNewOperations(t *testing.T) {
	operations := []Operation{"a/b/c", "x/y/z"}
	set := utils.NewSet(operations)
	t.Log(neatjsons.S(set))
}
