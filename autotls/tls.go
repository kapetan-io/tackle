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

package autotls

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/kapetan-io/tackle/set"
)

const (
	blockTypeEC   = "EC PRIVATE KEY"
	blockTypeRSA  = "RSA PRIVATE KEY"
	blockTypePriv = "PRIVATE KEY"
	blockTypeCert = "CERTIFICATE"
)

type Config struct {
	// (Optional) The path to the Trusted Certificate Authority.
	CaFile string

	// (Optional) The path to the Trusted Certificate Authority private key.
	CaKeyFile string

	// (Optional) The path to the un-encrypted key for the server certificate.
	KeyFile string

	// (Optional) The path to the server certificate.
	CertFile string

	// (Optional) If true will generate self-signed certificates. If CaFile and CaKeyFile
	//  is set but no KeyFile or CertFile is set then  will generate a self-signed key using
	//  the CaFile provided.
	AutoTLS bool

	// (Optional) Configures the MinVersion for ServerTLS. If not set, defaults to TLS 1.0
	MinVersion uint16

	// (Optional) Sets the Client Authentication type as defined in the 'tls' package.
	// Defaults to tls.NoClientCert.See the standard library tls.ClientAuthType for valid values.
	// If set to anything but tls.NoClientCert then Setup() attempts to load ClientAuthCaFile,
	// ClientAuthKeyFile and ClientAuthCertFile and sets those certs into the ClientTLS struct. If
	// none of the ClientXXXFile's are set, uses KeyFile and CertFile for client authentication.
	ClientAuth tls.ClientAuthType

	// (Optional) The path to the Trusted Certificate Authority used for client auth. If ClientAuth is
	// set and this field is empty, then CaFile is used to auth clients.
	ClientAuthCaFile string

	// (Optional) The path to the client private key, which is used to create the ClientTLS config. If
	// ClientAuth is set and this field is empty then KeyFile is used to create the ClientTLS.
	ClientAuthKeyFile string

	// (Optional) The path to the client cert key, which is used to create the ClientTLS config. If
	// ClientAuth is set and this field is empty then KeyFile is used to create the ClientTLS.
	ClientAuthCertFile string

	// (Optional) If InsecureSkipVerify is true, TLS clients will accept any certificate
	// presented by the server and any host name in that certificate.
	InsecureSkipVerify bool

	// (Optional) A Logger which implements the declared logger interface (typically *logrus.Entry)
	Logger StandardLogger

	// (Optional) The CA Certificate in PEM format. Used if CaFile is unset
	CaPEM *bytes.Buffer

	// (Optional) The CA Private Key in PEM format. Used if CaKeyFile is unset
	CaKeyPEM *bytes.Buffer

	// (Optional) The Certificate Key in PEM format. Used if KeyFile is unset.
	KeyPEM *bytes.Buffer

	// (Optional) The Certificate in PEM format. Used if CertFile is unset.
	CertPEM *bytes.Buffer

	// (Optional) The client auth CA Certificate in PEM format. Used if ClientAuthCaFile is unset.
	ClientAuthCaPEM *bytes.Buffer

	// (Optional) The client auth private key in PEM format. Used if ClientAuthKeyFile is unset.
	ClientAuthKeyPEM *bytes.Buffer

	// (Optional) The client auth Certificate in PEM format. Used if ClientAuthCertFile is unset.
	ClientAuthCertPEM *bytes.Buffer

	// (Optional) the server name to check when validating the provided certificate
	ClientAuthServerName string

	// ServerOrgName is the organization name used when generating a TLS certificate
	ServerOrgName string // TODO: Make sure there is a default

	// (Optional) The config created for use by the server. If set, all other
	// fields in this struct are ignored and this config is used. If unset, Setup()
	// will create a config using the above fields.
	ServerTLS *tls.Config

	// (Optional) The config created for use by clients and peer communication. If set, all other
	// fields in this struct are ignored and this config is used. If unset, Setup()
	// will create a config using the above fields.
	ClientTLS *tls.Config
}

func fromFile(name string) (*bytes.Buffer, error) {
	if name == "" {
		return nil, nil
	}

	b, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("while reading file '%s': %w", name, err)
	}
	return bytes.NewBuffer(b), nil
}

// Setup attempts to build a server and client TLS configuration given the Config provided.
func Setup(conf *Config) error {
	var err error

	if conf == nil {
		return nil
	}

	// If both client and server tls configs provided, nothing to do!
	if conf.ServerTLS != nil && conf.ClientTLS != nil {
		return nil
	}

	minServerTLSVersion := conf.MinVersion
	if minServerTLSVersion == 0 {
		minServerTLSVersion = tls.VersionTLS13
	}

	// Basic config with reasonably secure defaults
	set.Default(&conf.ServerTLS, &tls.Config{
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		ClientAuth: conf.ClientAuth,
		MinVersion: minServerTLSVersion,
		NextProtos: []string{
			"h2", "http/1.1", // enable HTTP/2
		},
	})
	set.Default(&conf.ClientTLS, &tls.Config{})

	// Attempt to load any files provided
	conf.CaPEM, err = fromFile(conf.CaFile)
	if err != nil {
		return err
	}

	conf.CaKeyPEM, err = fromFile(conf.CaKeyFile)
	if err != nil {
		return err
	}

	conf.KeyPEM, err = fromFile(conf.KeyFile)
	if err != nil {
		return err
	}

	conf.CertPEM, err = fromFile(conf.CertFile)
	if err != nil {
		return err
	}

	conf.ClientAuthCaPEM, err = fromFile(conf.ClientAuthCaFile)
	if err != nil {
		return err
	}

	conf.ClientAuthKeyPEM, err = fromFile(conf.ClientAuthKeyFile)
	if err != nil {
		return err
	}

	conf.ClientAuthCertPEM, err = fromFile(conf.ClientAuthCertFile)
	if err != nil {
		return err
	}

	set.Default(&conf.Logger, &NoOpLogger{})

	// If generated TLS certs requested
	if conf.AutoTLS {
		conf.Logger.Info("AutoTLS Enabled")
		set.Default(&conf.ServerOrgName, "Self Signed Org")

		// Generate CA Cert and Private Key
		if err := selfCA(conf); err != nil {
			return fmt.Errorf("while generating self signed CA certs: %w", err)
		}

		// Generate Server Cert and Private Key
		if err := selfCert(conf); err != nil {
			return fmt.Errorf("while generating self signed server certs: %w", err)
		}
	}

	if conf.CaPEM != nil {
		rootPool, err := x509.SystemCertPool()
		if err != nil {
			conf.Logger.Warn("while loading system CA Certs; using provided pool instead", "err", err)
			rootPool = x509.NewCertPool()
		}
		rootPool.AppendCertsFromPEM(conf.CaPEM.Bytes())
		conf.ServerTLS.RootCAs = rootPool
		conf.ClientTLS.RootCAs = rootPool
	}

	if conf.KeyPEM != nil && conf.CertPEM != nil {
		serverCert, err := tls.X509KeyPair(conf.CertPEM.Bytes(), conf.KeyPEM.Bytes())
		if err != nil {
			return fmt.Errorf("while parsing server certificate and private key: %w", err)
		}
		conf.ServerTLS.Certificates = []tls.Certificate{serverCert}
		conf.ClientTLS.Certificates = []tls.Certificate{serverCert}
	}

	// If user asked for client auth
	if conf.ClientAuth != tls.NoClientCert {
		clientPool := x509.NewCertPool()
		var certProvided bool
		if conf.ClientAuthCaPEM != nil {
			// If client auth CA was provided
			clientPool.AppendCertsFromPEM(conf.ClientAuthCaPEM.Bytes())
			certProvided = true

		} else if conf.CaPEM != nil {
			// else use the servers CA
			clientPool.AppendCertsFromPEM(conf.CaPEM.Bytes())
			certProvided = true
		}

		// error if neither cert was provided
		if !certProvided {
			return errors.New("client auth enabled, but no CA's provided")
		}

		conf.ServerTLS.ClientCAs = clientPool

		// If client auth key/cert was provided
		if conf.ClientAuthKeyPEM != nil && conf.ClientAuthCertPEM != nil {
			clientCert, err := tls.X509KeyPair(conf.ClientAuthCertPEM.Bytes(), conf.ClientAuthKeyPEM.Bytes())
			if err != nil {
				return fmt.Errorf("while parsing client certificate and private key: %w", err)
			}
			conf.ClientTLS.Certificates = []tls.Certificate{clientCert}
		}
	}

	conf.ClientTLS.ServerName = conf.ClientAuthServerName
	conf.ClientTLS.InsecureSkipVerify = conf.InsecureSkipVerify
	return nil
}

func selfCert(conf *Config) error {
	if conf.CertPEM != nil && conf.KeyPEM != nil {
		return nil
	}

	network, err := discoverNetwork()
	if err != nil {
		return fmt.Errorf("while detecting ip and host names: %w", err)
	}

	cert := x509.Certificate{
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		Subject:               pkix.Name{Organization: []string{conf.ServerOrgName}},
		NotAfter:              time.Now().Add(365 * (24 * time.Hour)),
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		SerialNumber:          big.NewInt(0xC0FFEE),
		NotBefore:             time.Now(),
		BasicConstraintsValid: true,
	}

	// Ensure all our names and ip addresses are included in the Certificate
	cert.DNSNames = append(cert.DNSNames, network.DNSNames...)

	for _, ipStr := range network.IPAddresses {
		if ip := net.ParseIP(ipStr); ip != nil {
			cert.IPAddresses = append(cert.IPAddresses, ip)
		}
	}

	conf.Logger.Info("Generating Server Private Key and Certificate....",
		"dns-names", strings.Join(cert.DNSNames, ","),
		"cert-ips", func() string {
			var r []string
			for i := range cert.IPAddresses {
				r = append(r, cert.IPAddresses[i].String())
			}
			return strings.Join(r, ",")
		}(),
	)

	// Generate a public / private key
	privKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return fmt.Errorf("while generating pubic/private key pair: %w", err)
	}

	// Attempt to sign the generated certs with the provided CaFile
	if conf.CaPEM == nil && conf.CaKeyPEM == nil {
		return errors.New("unable to generate server certs without a signing CA")
	}

	keyPair, err := tls.X509KeyPair(conf.CaPEM.Bytes(), conf.CaKeyPEM.Bytes())
	if err != nil {
		return fmt.Errorf("while reading generated PEMs: %w", err)
	}

	if len(keyPair.Certificate) == 0 {
		return errors.New("no certificates found in CA PEM")
	}

	caCert, err := x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return fmt.Errorf("while parsing CA Cert: %w", err)
	}

	signedBytes, err := x509.CreateCertificate(rand.Reader, &cert, caCert, &privKey.PublicKey, keyPair.PrivateKey)
	if err != nil {
		return fmt.Errorf("while self signing server cert: %w", err)
	}

	conf.CertPEM = new(bytes.Buffer)
	if err := pem.Encode(conf.CertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: signedBytes,
	}); err != nil {
		return fmt.Errorf("while encoding CERTIFICATE PEM: %w", err)
	}

	b, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("while encoding EC Marshalling: %w", err)
	}

	conf.KeyPEM = new(bytes.Buffer)
	if err := pem.Encode(conf.KeyPEM, &pem.Block{
		Type:  blockTypeEC,
		Bytes: b,
	}); err != nil {
		return fmt.Errorf("while encoding EC KEY PEM: %w", err)
	}
	return nil
}

func selfCA(conf *Config) error {
	ca := x509.Certificate{
		SerialNumber:          big.NewInt(2319),
		Subject:               pkix.Name{Organization: []string{conf.ServerOrgName}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	var privKey *ecdsa.PrivateKey
	var err error
	var b []byte

	if conf.CaPEM != nil && conf.CaKeyPEM != nil {
		return nil
	}

	conf.Logger.Info("Generating CA Certificates....")
	privKey, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return fmt.Errorf("while generating pubic/private key pair: %w", err)
	}

	b, err = x509.CreateCertificate(rand.Reader, &ca, &ca, &privKey.PublicKey, privKey)
	if err != nil {
		return fmt.Errorf("while self signing CA certificate: %w", err)
	}

	conf.CaPEM = new(bytes.Buffer)
	if err := pem.Encode(conf.CaPEM, &pem.Block{
		Type:  blockTypeCert,
		Bytes: b,
	}); err != nil {
		return fmt.Errorf("while encoding CERTIFICATE PEM: %w", err)
	}

	b, err = x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("while marshalling EC private key: %w", err)
	}

	conf.CaKeyPEM = new(bytes.Buffer)
	if err := pem.Encode(conf.CaKeyPEM, &pem.Block{
		Type:  blockTypeEC,
		Bytes: b,
	}); err != nil {
		return fmt.Errorf("while encoding EC private key into PEM: %w", err)
	}
	return nil
}

type netInfo struct {
	IPAddresses []string
	DNSNames    []string
}

// Attempts to discover all the external ips and dns names associated with the current host.
func discoverNetwork() (netInfo, error) {
	var result netInfo

	var err error
	result.IPAddresses, err = discoverNetworkAddresses()
	if err != nil {
		return result, err
	}

	for _, ip := range result.IPAddresses {
		records, _ := net.LookupAddr(ip)
		result.DNSNames = append(result.DNSNames, records...)
	}
	return result, nil
}

// Returns a list of net addresses by inspecting the network interfaces on the current host.
func discoverNetworkAddresses() ([]string, error) {
	var results []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			results = append(results, ip.String())
		}
	}
	return results, nil
}
