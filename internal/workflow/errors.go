package workflow

import "errors"

var ErrInvalidValidationPercentage = errors.New("real progress cannot exceed claimed progress")
