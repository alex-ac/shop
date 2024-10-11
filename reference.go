package shop

import (
	"fmt"
	"time"
)

type Reference struct {
	ApiVersion string        `json:"api_version"`
	Package    string        `json:"package"`
	Name       string        `json:"name"`
	Id         string        `json:"id"`
	UpdatedAt  UnixTimestamp `json:"updated_at"`
}

func NewReference(pkg, name, id string) (ref Reference, err error) {
	switch {
	case !IsValidPackageName(pkg):
		err = fmt.Errorf("%w: %s", ErrInvalidPackageName, pkg)
	case !IsValidRefName(name):
		err = fmt.Errorf("%w: %s", ErrInvalidReferenceName, name)
	case !IsValidInstanceId(id):
		err = fmt.Errorf("%w: %s", ErrInvalidInstanceId, id)
	default:
		ref = Reference{
			ApiVersion: LatestVersion,
			Package:    pkg,
			Name:       name,
			Id:         id,
			UpdatedAt:  UnixTimestamp{time.Now()},
		}
	}
	return
}

func IsValidRefName(v string) bool {
	// refs & tags has the same name format.
	return IsValidTagName(v)
}
