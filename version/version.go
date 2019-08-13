package version

import "fmt"

// Type is a binary version struct
type Type struct {
	Version string
	Commit  string
	Date    string
}

func (t Type) String() string {
	return fmt.Sprintf("%s-%s-%s", t.Version, t.Commit, t.Date)
}

// Info is a global variable that holds binary version info
var Info Type
