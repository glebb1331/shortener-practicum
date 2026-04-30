package tlscert

import (
	"crypto/x509"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelfSignedTLSConfig(t *testing.T) {
	cfg, err := SelfSignedTLSConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Certificates, 1)

	rawCert := cfg.Certificates[0].Certificate
	require.NotEmpty(t, rawCert)

	cert, err := x509.ParseCertificate(rawCert[0])
	require.NoError(t, err)

	assert.Contains(t, cert.DNSNames, "localhost")
	assert.NotEmpty(t, cert.IPAddresses)
	assert.Equal(t, "Shortener", cert.Subject.Organization[0])
	assert.True(t, cert.NotAfter.After(cert.NotBefore))
}
