package k8sfips

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// CreateFIPSCompliantTLSConfig creates a TLS configuration with FIPS 140-2 approved settings
func CreateFIPSCompliantTLSConfig() *tls.Config {
	return &tls.Config{
		// Use FIPS 140-2 approved cipher suites only
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		},
		// Use FIPS-approved elliptic curves only
		CurvePreferences: []tls.CurveID{
			tls.CurveP256, // NIST P-256 (FIPS approved)
			tls.CurveP384, // NIST P-384 (FIPS approved)
		},
		MinVersion: tls.VersionTLS12, // TLS 1.2 minimum (FIPS requirement)
		MaxVersion: tls.VersionTLS13, // TLS 1.3 allowed
	}
}

// NewFIPSCompliantClient creates a FIPS-compliant Kubernetes client
func NewFIPSCompliantClient(cfg *rest.Config) (client.Client, error) {
	// Create a copy of the config to avoid modifying the original
	configCopy := rest.CopyConfig(cfg)

	// Create FIPS-compliant TLS config
	tlsConfig := CreateFIPSCompliantTLSConfig()

	// Preserve server name if set
	if configCopy.TLSClientConfig.ServerName != "" {
		tlsConfig.ServerName = configCopy.TLSClientConfig.ServerName
	}

	// Preserve insecure setting if set
	if configCopy.TLSClientConfig.Insecure {
		tlsConfig.InsecureSkipVerify = true
	}

	// Clear ALL TLS config fields to avoid conflicts with custom transport
	configCopy.TLSClientConfig = rest.TLSClientConfig{}

	// Set our custom transport
	configCopy.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	httpClient := &http.Client{
		Transport: configCopy.Transport,
		Timeout:   30 * time.Second,
	}

	mapper, err := apiutil.NewDynamicRESTMapper(configCopy, httpClient)
	if err != nil {
		err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, false, func(ctx context.Context) (bool, error) {
			mapper, err = apiutil.NewDynamicRESTMapper(configCopy, httpClient)
			if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return nil, fmt.Errorf("while waiting for client mapper: %w", err)
		}
	}

	cli, err := client.New(configCopy, client.Options{Mapper: mapper})
	if err != nil {
		return nil, fmt.Errorf("while creating a client: %w", err)
	}
	return cli, nil
}

// NewFIPSCompliantDynamicClient creates a FIPS-compliant dynamic Kubernetes client
func NewFIPSCompliantDynamicClient(cfg *rest.Config) (dynamic.Interface, error) {
	// Create a copy of the config to avoid modifying the original
	configCopy := rest.CopyConfig(cfg)

	// Create FIPS-compliant TLS config
	tlsConfig := CreateFIPSCompliantTLSConfig()

	// Preserve server name if set
	if configCopy.TLSClientConfig.ServerName != "" {
		tlsConfig.ServerName = configCopy.TLSClientConfig.ServerName
	}

	// Preserve insecure setting if set
	if configCopy.TLSClientConfig.Insecure {
		tlsConfig.InsecureSkipVerify = true
	}

	// Clear ALL TLS config fields to avoid conflicts with custom transport
	configCopy.TLSClientConfig = rest.TLSClientConfig{}

	// Set our custom transport
	configCopy.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return dynamic.NewForConfig(configCopy)
}
