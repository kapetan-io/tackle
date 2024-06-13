# Tackle
Kapetan's tackle box of libraries for golang.

## SET config values
Simplify setting default values during configuration.
```go
import "github.com/kapetan-io/tackle/set"

var config struct {
    Bang float64
    Foo  string
    Bar  int
}

// Supply additional default values and set.Default() will
// choose the first default that is not of zero value
set.Default(&config.Foo, os.Getenv("FOO"), "default")
// Sets Bar to 200 if Bar is not already set
set.Default(&config.Bar, 200)
// Sets Bang to the environment variable "BANG" if set, else sets the
// value to 5.0
set.Default(&config.Bang, set.EnvNumber[float64]("BANG"), 5.0)

// The following is equivalent to the above set.Default() calls
// and demonstrates the complexity reduced by using set.Default()
if config.Foo == "" {
    config.Foo = os.Getenv("FOO")
    if config.Foo == "" {
        config.Foo = "default"
    }
}
if config.Bar == 0 {
    config.Bar = 200
}
if config.Bang == 0.0 {
    v, _ := strconv.ParseFloat(os.Getenv("BANG"), 64)
    config.Bang = float64(v)
    if config.Bang == 0.0 {
        config.Bang = 5.0
    }
}

// Use set.Override() to assign the first value that is not empty or of zero
// value. 
loadFromFile(&config)
argFoo = flag.String("foo", "", "foo via cli arg")

// The following will override the config file if 'foo' is provided via
// the cli or defined in the environment.
set.Override(&config.Foo, *argFoo, os.Env("FOO"))

// Returns true if 'value' is zero (the default golang value)
var value string
set.IsZero(value)

// Returns true if 'value' is zero (the default golang value)
set.IsZeroValue(reflect.ValueOf(value))
```