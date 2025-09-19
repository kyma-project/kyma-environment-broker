package httputil

import (
	"crypto/tls"
	"net/http"
	"time"
)

// FIPSCompliantTLSConfig returns a TLS configuration that complies with FIPS 140-2 requirements
func FIPSCompliantTLSConfig() *tls.Config {
	return &tls.Config{
		// Use only FIPS-approved cipher suites
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		},
		// Use only FIPS-approved elliptic curves (exclude X25519)
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.CurveP384,
			tls.CurveP521,
		},
		// Require TLS 1.2 minimum
		MinVersion: tls.VersionTLS12,
	}
}

// NewFIPSCompliantClient creates an HTTP client with FIPS-compliant TLS configuration
func NewFIPSCompliantClient(timeoutSec time.Duration, skipCertVerification bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = FIPSCompliantTLSConfig()
	transport.TLSClientConfig.InsecureSkipVerify = skipCertVerification

	return &http.Client{
		Transport: transport,
		Timeout:   timeoutSec * time.Second,
	}
}
