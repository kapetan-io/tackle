/*
Copyright 2024 Derrick J. Wippler

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

package autotls_test

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/kapetan-io/tackle/autotls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log"
	"net/http"
	"sync"
	"testing"
)

func TestSetup(t *testing.T) {
	tests := []struct {
		tls  *autotls.Config
		name string
	}{
		{
			name: "user provided certificates",
			tls: &autotls.Config{
				CaFile:   "certs/ca.cert",
				CertFile: "certs/auto.pem",
				KeyFile:  "certs/auto.key",
			},
		},
		{
			name: "user provided certificate without IP SANs",
			tls: &autotls.Config{
				CaFile:               "certs/ca.cert",
				CertFile:             "certs/auto_no_ip_san.pem",
				KeyFile:              "certs/auto_no_ip_san.key",
				ClientAuthServerName: "auto",
			},
		},
		{
			name: "auto tls",
			tls: &autotls.Config{
				AutoTLS: true,
			},
		},
		{
			name: "generate server certs with user provided ca",
			tls: &autotls.Config{
				CaFile:    "certs/ca.cert",
				CaKeyFile: "certs/ca.key",
				AutoTLS:   true,
			},
		},
		{
			name: "client auth enabled",
			tls: &autotls.Config{
				CaFile:     "certs/ca.cert",
				CaKeyFile:  "certs/ca.key",
				AutoTLS:    true,
				ClientAuth: tls.RequireAndVerifyClientCert,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := autotls.Setup(tt.tls)
			require.NoError(t, err)

			srv := http.Server{
				Addr: "localhost:9685",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = fmt.Fprintln(w, "Hello, client")
				}),
				TLSConfig: tt.tls.ServerTLS,
			}
			defer func() { _ = srv.Close() }()

			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				err = srv.ListenAndServeTLS("", "")
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					t.Logf("server listen error: %v", err)
				}
			}()

			c := &http.Client{
				Transport: &http.Transport{TLSClientConfig: tt.tls.ClientTLS},
			}

			resp, err := c.Get("https://localhost:9685/")
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			b, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, "Hello, client\n", string(b))
			_ = srv.Shutdown(context.Background())
			wg.Wait()
		})
	}
}

func TestSetupTLSSkipVerify(t *testing.T) {

	// Use existing TLS Certs for the server
	var serverTLS = autotls.Config{
		CaFile:   "certs/ca.cert",
		CertFile: "certs/auto.pem",
		KeyFile:  "certs/auto.key",
	}
	err := autotls.Setup(&serverTLS)
	require.NoError(t, err)

	srv := http.Server{
		Addr: "localhost:9685",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, "Hello, client")
		}),
		TLSConfig: serverTLS.ServerTLS,
	}
	defer func() { _ = srv.Close() }()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = srv.ListenAndServeTLS("", "")
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Logf("server listen error: %v", err)
		}
	}()

	// Auto generate TLS certs and tell TLS to skip the domain verification step
	var clientTLS = autotls.Config{
		AutoTLS:            true,
		InsecureSkipVerify: true,
	}
	err = autotls.Setup(&clientTLS)
	require.NoError(t, err)

	c := &http.Client{
		Transport: &http.Transport{TLSClientConfig: clientTLS.ClientTLS},
	}

	resp, err := c.Get("https://localhost:9685/")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "Hello, client\n", string(b))
	_ = srv.Shutdown(context.Background())
	wg.Wait()
}

func TestSetupTLSClientAuth(t *testing.T) {
	serverTLS := autotls.Config{
		CaFile:           "certs/ca.cert",
		CertFile:         "certs/auto.pem",
		KeyFile:          "certs/auto.key",
		ClientAuthCaFile: "certs/client-auth-ca.pem",
		ClientAuth:       tls.RequireAndVerifyClientCert,
	}
	err := autotls.Setup(&serverTLS)
	require.NoError(t, err)

	srv := http.Server{
		Addr: "localhost:9685",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, "Hello, client")
		}),
		ErrorLog:  log.New(io.Discard, "", log.LstdFlags),
		TLSConfig: serverTLS.ServerTLS,
	}
	defer func() { _ = srv.Close() }()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = srv.ListenAndServeTLS("", "")
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Logf("server listen error: %v", err)
		}
	}()

	// Given generated client certs
	clientTLS := &autotls.Config{
		AutoTLS:            true,
		InsecureSkipVerify: true,
	}
	require.NoError(t, autotls.Setup(clientTLS))

	// Should NOT be allowed without a cert signed by the client CA
	c := &http.Client{
		Transport: &http.Transport{TLSClientConfig: clientTLS.ClientTLS},
	}

	_, err = c.Get("https://localhost:9685/")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tls: certificate required")

	// Given the client auth certs
	clientTLS = &autotls.Config{
		CertFile:           "certs/client-auth.pem",
		KeyFile:            "certs/client-auth.key",
		InsecureSkipVerify: true,
	}
	require.NoError(t, autotls.Setup(clientTLS))

	c = &http.Client{
		Transport: &http.Transport{TLSClientConfig: clientTLS.ClientTLS},
	}

	resp, err := c.Get("https://localhost:9685/")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "Hello, client\n", string(b))
	_ = srv.Shutdown(context.Background())
	wg.Wait()

}
