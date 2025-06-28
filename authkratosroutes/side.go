package authkratosroutes

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/orzkratos/authkratos/internal/utils"
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

func (c *SelectPath) Match(operation string) bool {
	switch c.SelectSide {
	case INCLUDE:
		if c.Operations == nil {
			return false
		}
		return c.Operations[Path(operation)]
	case EXCLUDE:
		if c.Operations == nil {
			return true
		}
		return !c.Operations[Path(operation)]
	default:
		panic(c.SelectSide)
	}
}

func (c *SelectPath) NewMatchFunc(description string, logger log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(logger)

	return func(ctx context.Context, operation string) bool {
		match := c.Match(operation)
		if match {
			LOG.Debugf("operation=%s include=%v match=%d next -> %s", operation, c.SelectSide, utils.BooleanToNum(match), description)
		} else {
			LOG.Debugf("operation=%s include=%v match=%d skip -- %s", operation, c.SelectSide, utils.BooleanToNum(match), description)
		}
		return match
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
