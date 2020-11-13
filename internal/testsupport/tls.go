package testsupport

// Copied and adapted from https://github.com/jsha/minica rev
// e81e95a9e94be80e9da3c1e2b55accdd4884128b which cannot be used as a library.

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"math"
	"math/big"
	"testing"
	"time"
)

const rsaBits = 2048

type issuer struct {
	key  crypto.Signer
	cert *x509.Certificate
}

func makeIssuer(t *testing.T) *issuer {
	const op = "testsupport/makeIssuer"
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	cert, err := makeRootCert(key)
	if err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	return &issuer{key: key, cert: cert}
}

func makeKey(t *testing.T) *rsa.PrivateKey {
	const op = "testsupport/makeKey"
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	return key
}

func makeRootCert(key crypto.Signer) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	skid, err := calculateSKID(key.Public())
	if err != nil {
		return nil, err
	}
	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: "minica root ca " + hex.EncodeToString(serial.Bytes()[:3]),
		},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(100, 0, 0),

		SubjectKeyId:          skid,
		AuthorityKeyId:        skid,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(der)
}

func calculateSKID(pubKey crypto.PublicKey) ([]byte, error) {
	spkiASN1, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	var spki struct {
		Algorithm        pkix.AlgorithmIdentifier
		SubjectPublicKey asn1.BitString
	}
	_, err = asn1.Unmarshal(spkiASN1, &spki)
	if err != nil {
		return nil, err
	}
	skid := sha1.Sum(spki.SubjectPublicKey.Bytes)
	return skid[:], nil
}

func sign(t *testing.T, iss *issuer, cn string) (*rsa.PrivateKey, *x509.Certificate) {
	const op = "testsupport/sign"
	t.Helper()

	if cn == "" {
		t.Fatalf("%s: no common name", op)
	}

	key := makeKey(t)
	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		t.Fatalf("%s: %v", op, err)
	}

	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: cn,
		},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 0, 30),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, iss.cert, key.Public(), iss.key)
	if err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	return key, cert
}

func encodePrivateKey(t *testing.T, key *rsa.PrivateKey) string {
	const op = "testsupport/encodePrivateKey"
	var buf bytes.Buffer

	t.Helper()

	bs := x509.MarshalPKCS1PrivateKey(key)
	err := pem.Encode(&buf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: bs,
	})
	if err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	return buf.String()
}

func encodeCert(t *testing.T, cert *x509.Certificate) string {
	const op = "testsupport/encodeCert"
	var buf bytes.Buffer
	t.Helper()

	err := pem.Encode(&buf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
	if err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	return buf.String()
}

// TLSPair represents a TLS key and certificate pair.
type TLSPair struct {
	Key  string
	Cert string
}

// NewTLSPair generates new pair of TLS certificate and private key for the
// passed common name.
func NewTLSPair(t *testing.T, cn string) *TLSPair {
	t.Helper()

	iss := makeIssuer(t)
	key, cert := sign(t, iss, cn)
	return &TLSPair{
		Key:  encodePrivateKey(t, key),
		Cert: encodeCert(t, cert),
	}
}
