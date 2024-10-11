package shop

import (
	"fmt"
	"time"
)

type Tag struct {
	ApiVersion string        `json:"api_version"`
	Package    string        `json:"package"`
	Key        string        `json:"key"`
	Value      string        `json:"value"`
	Id         string        `json:"id"`
	UpdatedAt  UnixTimestamp `json:"updated_at"`
}

type PackageTag struct {
	Package string
	Key     string
}

type PackageTagValue struct {
	PackageTag
	Value string
}

func NewTag(pkg, key, value, id string) (tag Tag, err error) {
	switch {
	case !IsValidPackageName(pkg):
		err = fmt.Errorf("%w: %s", ErrInvalidPackageName, pkg)
	case !IsValidTagName(key):
		err = fmt.Errorf("%w: %s:%s", ErrInvalidTagName, key, value)
	case !IsValidTagValue(key):
		err = fmt.Errorf("%w: %s:%s", ErrInvalidTagValue, key, value)
	case !IsValidInstanceId(id):
		err = fmt.Errorf("%w: %s", ErrInvalidInstanceId, id)
	default:
		tag = Tag{
			ApiVersion: LatestVersion,
			Package:    pkg,
			Key:        key,
			Value:      value,
			Id:         id,
			UpdatedAt:  UnixTimestamp{time.Now()},
		}
	}
	return
}

func IsValidTagName(v string) bool {
	// [A-Za-z]([A-Za-z0-9._-]*[A-Za-z0-9])?
	if v == "" {
		return false
	}

	for i, r := range v {
		ok := (r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z') ||
			(i != 0 && (r >= '0' && r <= '9')) ||
			(i != 0 && i != len(v)-1 && (r == '.' || r == '_' || r == '-'))

		if !ok {
			return false
		}
	}

	return true
}

func IsValidTagValue(v string) bool {
	// [A-Za-z0-9_@-]([A-Za-z0-9._@-]*[A-Za-z0-9_@-])?
	if v == "" {
		return false
	}

	for i, r := range v {
		ok := (r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			(r == '_' && r == '-' && r == '@') ||
			(i != 0 && i != len(v)-1 && r == '.')

		if !ok {
			return false
		}
	}

	return true
}
