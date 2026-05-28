package llm

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeCertPair generates a throwaway CA + client cert/key and writes them as
// PEM files, returning their paths. Mirrors what infra/mtls/gen-certs.sh
// produces, so buildMTLSConfig is exercised without shelling out to openssl.
func writeCertPair(t *testing.T) (caPath, certPath, keyPath string) {
	t.Helper()
	dir := t.TempDir()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "api-client"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
	require.NoError(t, err)

	leafKeyDER, err := x509.MarshalECPrivateKey(leafKey)
	require.NoError(t, err)

	caPath = filepath.Join(dir, "ca.pem")
	certPath = filepath.Join(dir, "client.crt")
	keyPath = filepath.Join(dir, "client.key")
	writePEM(t, caPath, "CERTIFICATE", caDER)
	writePEM(t, certPath, "CERTIFICATE", leafDER)
	writePEM(t, keyPath, "EC PRIVATE KEY", leafKeyDER)
	return caPath, certPath, keyPath
}

func writePEM(t *testing.T, path, blockType string, der []byte) {
	t.Helper()
	b := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
	require.NoError(t, os.WriteFile(path, b, 0o600))
}

func TestBuildMTLSConfig_LoadsCerts(t *testing.T) {
	ca, crt, key := writeCertPair(t)
	cfg := &Config{MTLSEnabled: true, MTLSCAPath: ca, MTLSCertPath: crt, MTLSKeyPath: key}

	tlsCfg, err := buildMTLSConfig(cfg)
	require.NoError(t, err)
	require.Len(t, tlsCfg.Certificates, 1)
	assert.NotNil(t, tlsCfg.RootCAs)
	assert.Equal(t, uint16(tls.VersionTLS12), tlsCfg.MinVersion)
}

func TestBuildMTLSConfig_MissingPaths(t *testing.T) {
	_, err := buildMTLSConfig(&Config{MTLSEnabled: true})
	require.Error(t, err)
}

func TestBuildMTLSConfig_BadCAFile(t *testing.T) {
	_, crt, key := writeCertPair(t)
	dir := t.TempDir()
	badCA := filepath.Join(dir, "ca.pem")
	require.NoError(t, os.WriteFile(badCA, []byte("not a pem"), 0o600))

	_, err := buildMTLSConfig(&Config{MTLSEnabled: true, MTLSCAPath: badCA, MTLSCertPath: crt, MTLSKeyPath: key})
	require.Error(t, err)
}
