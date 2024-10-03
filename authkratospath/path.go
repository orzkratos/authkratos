package authkratospath

type Path string

func New(path string) Path {
	return Path(path)
}

type Paths []Path

func NewPathsMap(paths []Path) map[Path]bool {
	var mp = make(map[Path]bool, len(paths))
	for _, path := range paths {
		mp[path] = true
	}
	return mp
}