package authkratosroutes

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
		Operations: NewPathsBooMap(paths),
	}
}

func NewExclude(paths ...Path) *SelectPath {
	return &SelectPath{
		SelectSide: EXCLUDE,
		Operations: NewPathsBooMap(paths),
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
