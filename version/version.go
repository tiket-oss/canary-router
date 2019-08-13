package version

import "fmt"

type Type struct {
	Version string
	Commit  string
	Date    string
}

func (t Type) String() string {
	return fmt.Sprintf("%s-%s-%s", t.Version, t.Commit, t.Date)
}

var Info Type
