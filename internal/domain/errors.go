package domain

import "errors"

var (
	ErrDuplicateProjectName = errors.New("Duplicate Project Name")
	ErrProjectNotFound      = errors.New("Project Not Found")
	ErrNotFound             = errors.New("Resource Not Found")
	ErrConflict             = errors.New("Resource Already Exists")
	ErrInvalidRequest       = errors.New("Invalid Request")
	ErrUnauthorized         = errors.New("Unauthorized")
	ErrUnavailable          = errors.New("Service Unavailable")
)
