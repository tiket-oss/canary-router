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

// ShortHash returns first 7 chars of Git commit hash
func ShortHash(hash string) string {
	if len(hash) >= 7 {
		return hash[:7]
	}

	return hash
}

// Info is a global variable that holds binary version info
var Info Type
