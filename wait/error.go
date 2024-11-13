package wait

import "fmt"

// MultiError encapsulates multiple errors.
//
// Borrowed from the App Engine SDK.
type MultiError []error

func (e MultiError) Error() string {
	if len(e) == 0 {
		return ""
	}

	var result string
	for _, err := range e {
		result = err.Error()
		break
	}
	switch len(e) {
	case 0:
		return "(0 errors)"
	case 1:
		return result
	case 2:
		return result + " (and 1 other error)"
	}
	return fmt.Sprintf("%s (and %d other errors)", result, len(e)-1)
}
