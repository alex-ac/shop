package shop

import (
	"errors"
)

var (
	ErrUnimplemented             = errors.New("Unimplemented")
	ErrRegistryAdminIsNotAllowed = errors.New("Admin action on the registry is not enabled in configuration")
	ErrRegistryWriteIsNotAllowed = errors.New("Write action on the registry is not enabled in configuration")
	ErrUnknownRepo               = errors.New("Registry does not have repo")
	ErrInvalidPackageName        = errors.New("Invalid package name")
	ErrInvalidInstanceId         = errors.New("Invalid instance id")
	ErrInvalidReferenceName      = errors.New("Invalid reference name")
	ErrInvalidTagName            = errors.New("Invalid tag name")
	ErrInvalidTagValue           = errors.New("Invalid tag value")
)
