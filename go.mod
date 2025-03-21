module github.com/kapetan-io/tackle

// tackle requires 1.22 or greater. 1.22 fixes closure scope issues with previous golang versions.
// This is required for proper use of `wait.Group` and `wait.FanOut`.
// See https://go.dev/blog/loopvar-preview for details
go 1.23.0

toolchain go1.23.7

require (
	github.com/stretchr/testify v1.9.0
	golang.org/x/net v0.37.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
