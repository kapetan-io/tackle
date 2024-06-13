package set

import (
	"os"
	"reflect"
	"strconv"
)

type constraints interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 |
		~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64
}

// EnvNumber retrieves the value of the environment variable named by the key
// and then preforms a string conversion using the `strconv` library to parse
// into the requested integer type. If the retrieved string value fails to parse into
// the requested integer type, returns the zero value of that type.
func EnvNumber[T constraints](key string) T {
	s := os.Getenv(key)
	var z T
	rt := reflect.TypeOf(z)
	switch rt.Kind() {
	case reflect.Float32, reflect.Float64:
		t, err := strconv.ParseFloat(s, rt.Bits())
		if err != nil {
			return T(0)
		}
		return T(t)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		t, err := strconv.ParseInt(s, 10, rt.Bits())
		if err != nil {
			return z
		}
		return T(t)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		t, err := strconv.ParseUint(s, 10, rt.Bits())
		if err != nil {
			return z
		}
		return T(t)
	default:
		return z
	}
}
