/*
Copyright 2024 Kapetan.io Technologies

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package set_test

import (
	"github.com/kapetan-io/tackle/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strconv"
	"testing"
)

func TestIfEmpty(t *testing.T) {
	var conf struct {
		Foo string
		Bar int
	}
	assert.Equal(t, "", conf.Foo)
	assert.Equal(t, 0, conf.Bar)

	// Should apply the default values
	set.Default(&conf.Foo, "default")
	set.Default(&conf.Bar, 200)

	assert.Equal(t, "default", conf.Foo)
	assert.Equal(t, 200, conf.Bar)

	conf.Foo = "thrawn"
	conf.Bar = 500

	// Should NOT apply the default values
	set.Default(&conf.Foo, "default")
	set.Default(&conf.Bar, 200)

	assert.Equal(t, "thrawn", conf.Foo)
	assert.Equal(t, 500, conf.Bar)
}

func TestIfDefaultPrecedence(t *testing.T) {
	var conf struct {
		Foo string
		Bar string
	}
	assert.Equal(t, "", conf.Foo)
	assert.Equal(t, "", conf.Bar)

	// Should use the final default value
	envValue := ""
	set.Default(&conf.Foo, envValue, "default")
	assert.Equal(t, "default", conf.Foo)

	// Should use envValue
	envValue = "bar"
	set.Default(&conf.Bar, envValue, "default")
	assert.Equal(t, "bar", conf.Bar)
}

func TestIsEmpty(t *testing.T) {
	var count64 int64
	var thing string

	// Should return true
	assert.Equal(t, true, set.IsZero(count64))
	assert.Equal(t, true, set.IsZero(thing))

	thing = "thrawn"
	count64 = int64(1)
	assert.Equal(t, false, set.IsZero(count64))
	assert.Equal(t, false, set.IsZero(thing))
}

func TestIfEmptyTypePanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			assert.Equal(t, "reflect.Set: value of type int is not assignable to type string", r)
		}
	}()

	var thing string
	// Should panic
	set.Default(&thing, 1)
	assert.Fail(t, "Should have caught panic")
}

func TestIfEmptyNonPtrPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			assert.Equal(t, "set.Default: Expected first argument to be of type reflect.Ptr", r)
		}
	}()

	var thing string
	// Should panic
	set.Default(thing, "thrawn")
	assert.Fail(t, "Should have caught panic")
}

type MyInterface interface {
	Thing() string
}

type MyImplementation struct{}

func (s *MyImplementation) Thing() string {
	return "thing"
}

func NewImplementation() MyInterface {
	// Type and Value are not nil
	var p *MyImplementation = nil
	return p
}

type MyStruct struct {
	T MyInterface
}

func NewMyStruct(t MyInterface) *MyStruct {
	return &MyStruct{T: t}
}

func TestIsNil(t *testing.T) {
	m := MyStruct{T: &MyImplementation{}}
	assert.True(t, m.T != nil)
	m.T = nil
	assert.True(t, m.T == nil)

	o := NewMyStruct(nil)
	assert.True(t, o.T == nil)

	thing := NewImplementation()
	assert.False(t, thing == nil)
	assert.True(t, set.IsNil(thing))
	assert.False(t, set.IsNil(&MyImplementation{}))
}

func TestEnvNumber(t *testing.T) {
	require.NoError(t, os.Setenv("INTEGER", "1"))
	require.NoError(t, os.Setenv("FLOAT", "1.0"))

	assert.Equal(t, int64(1), set.EnvNumber[int64]("INTEGER"))
	assert.Equal(t, int32(1), set.EnvNumber[int32]("INTEGER"))
	assert.Equal(t, float32(1.0), set.EnvNumber[float32]("FLOAT"))
	assert.Equal(t, float64(1.0), set.EnvNumber[float64]("FLOAT"))
}

func TestExample(t *testing.T) {
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
	// Sets Bang to the environment variable "BANG" if set else 5.0
	set.Default(&config.Bang, set.EnvNumber[float64]("BANG"), 5.0)

	// The following is equivalent to the above set.Default() calls
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

	assert.Equal(t, config.Bang, 5.0)
	assert.Equal(t, config.Foo, "default")
	assert.Equal(t, config.Bar, 200)
}
