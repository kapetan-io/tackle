## AutoTLS
Provides functions which can generate TLS certificates suitable for both client and server certificates.
The package can configure a ready to use `*tls.Config` for both client and server depending on the 
`autotls.Config` provided.

With Support For
- User provided certificate files `*.pem`
- Client side TLS configs
- Automatic generation of valid client and server TLS configs
- Client side TLS authentication
- Skip insecure

See `autotls.Config` for all available options.

```go
package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
    
	"github.com/kapetan-io/tackle/autotls"
)

func main() {
    tls := autotls.Config{
        // Optionally provide your own certificates
        //CaFile:           "certs/ca.cert",
        //CertFile:         "certs/auto.pem",
        //KeyFile:          "certs/auto.key",

        // Optionally setup client side cert authentication
        //ClientAuthCaFile: "certs/client-auth-ca.pem",
        //ClientAuth:       tls.RequireAndVerifyClientCert,

        // Generate both client and server certs on the fly
        AutoTLS: true,
    }

    err := autotls.Setup(&tls)
    if err != nil {
        panic(err)
    }

    // After Setup() returns without error `tls` will be populated with relevant TLS config information
    // tls.ServerTLS <-- the server certificates
    // tls.ClientTLS <-- the client certificates

    srv := http.Server{
        Addr: "localhost:9685",
        Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            _, _ = fmt.Fprintln(w, "Hello, client")
        }),
        // ServerTLS is the TLS certificate for use by the server
        TLSConfig: tls.ServerTLS,
    }
    defer srv.Close()

    go func() {
        // CertFile and KeyFile are provided via autotls.ServerTLS
        err = srv.ListenAndServeTLS("", "")
        if err != nil && !errors.Is(http.ErrServerClosed, err) {
            fmt.Printf("server listen error: %v", err)
        }
    }()

    c := &http.Client{
        // TLSClientConfig is the client side TLS certificate
        Transport: &http.Transport{TLSClientConfig: tls.ClientTLS},
    }

    resp, err := c.Get("https://localhost:9685/")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    b, err := io.ReadAll(resp.Body)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Client: %s\n", string(b))
}
```