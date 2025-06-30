package authkratosroutes

import (
	"golang.org/x/exp/maps"
)

type SelectSide string

const (
	INCLUDE SelectSide = "INCLUDE"
	EXCLUDE SelectSide = "EXCLUDE"
)

type SelectPath struct {
	SelectSide SelectSide
	Operations map[Path]bool
}

func NewInclude(paths ...Path) *SelectPath {
	return &SelectPath{
		SelectSide: INCLUDE,
		Operations: NewOperations(paths),
	}
}

func NewExclude(paths ...Path) *SelectPath {
	return &SelectPath{
		SelectSide: EXCLUDE,
		Operations: NewOperations(paths),
	}
}

func (c *SelectPath) Match(operation Path) bool {
	switch c.SelectSide {
	case INCLUDE:
		if c.Operations == nil {
			return false
		}
		return c.Operations[operation]
	case EXCLUDE:
		if c.Operations == nil {
			return true
		}
		return !c.Operations[operation]
	default:
		panic(c.SelectSide)
	}
}

func (c *SelectPath) Opposite() *SelectPath {
	switch c.SelectSide {
	case INCLUDE:
		return NewExclude(maps.Keys(c.Operations)...)
	case EXCLUDE:
		return NewInclude(maps.Keys(c.Operations)...)
	default:
		panic("unknown select-side: " + string(c.SelectSide))
	}
}
