package utils

import (
	"testing"

	"github.com/yyle88/neatjson/neatjsons"
)

func TestMpKB(t *testing.T) {
	t.Log(neatjsons.S(MpKB([]string{"a", "b", "c"})))
}

func TestSample(t *testing.T) {
	t.Log(neatjsons.S(Sample([]string{"a", "b", "c"})))
}
