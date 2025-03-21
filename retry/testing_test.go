package retry_test

import (
	"fmt"
	"github.com/kapetan-io/tackle/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/nettest"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestUntilConnect(t *testing.T) {
	ln, err := nettest.NewLocalListener("tcp")
	require.NoError(t, err)

	go func() {
		cn, err := ln.Accept()
		require.NoError(t, err)
		cn.Close()
	}()
	// Wait until we can connect, then continue with the test
	retry.UntilConnect(t, 10, time.Millisecond*100, ln.Addr().String())
}

func TestUntilPass(t *testing.T) {
	var value string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Sleep some rand amount to time to simulate some
			// async process happening on the server
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			// Set the value
			value = r.FormValue("value")
		} else {
			fmt.Fprintln(w, value)
		}
	}))
	defer ts.Close()

	// Start the async process that produces a value on the server
	res, err := http.PostForm(ts.URL, url.Values{"value": []string{"batch job completed"}})
	require.NoError(t, err)
	defer func() {
		err := res.Body.Close()
		require.NoError(t, err)
	}()

	// Keep running this until the batch job completes or attempts are exhausted
	retry.UntilPass(t, 10, time.Millisecond*100, func(t retry.TestingT) {
		r, err := http.Get(ts.URL)

		// use of `require` will abort the current test here and tell UntilPass() to
		// run again after 100 milliseconds
		require.NoError(t, err)

		defer func() {
			err = r.Body.Close()
			require.NoError(t, err)
		}()

		// Or you can check if the assert returned true or not
		if !assert.Equal(t, 200, r.StatusCode) {
			return
		}

		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		assert.Equal(t, "batch job completed\n", string(b))
	})
}
