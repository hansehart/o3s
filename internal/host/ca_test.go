package host

import (
	"bytes"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

// rfc3526Group14 is the 2048-bit MODP prime of RFC 3526, section 3, as the RFC prints it.
const rfc3526Group14 = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF"

// aCA makes a CA in a folder of its own, with a key small enough for a quick test.
func aCA(t *testing.T) string {
	t.Helper()
	old := keyBits
	keyBits = 1024
	t.Cleanup(func() { keyBits = old })
	dir := filepath.Join(t.TempDir(), "o3s")
	// Stop when the CA stays unmade
	if _, err := CA(dir); err != nil {
		t.Fatalf("CA() = %v", err)
	}
	return dir
}

// The CA lands in a folder for its owner alone, the key for its owner and the public cert for every reader.
func TestCAWritesItsFilesWithTheirModes(t *testing.T) {
	dir := aCA(t)

	modes := map[string]os.FileMode{".": 0o700, "mitmproxy-ca.pem": 0o600, "mitmproxy-ca.crt": 0o644, "mitmproxy-dhparam.pem": 0o600}

	// Check every file's mode
	for name, want := range modes {
		info, err := os.Stat(filepath.Join(dir, name))
		// Report a file that is missing or open to others than it should be
		if err != nil || info.Mode().Perm() != want {
			t.Errorf("%s: %v, mode %v, want %v", name, err, info.Mode().Perm(), want)
		}
	}
}

// The cert is a CA that signs the proxy's leaf certs and nothing below them.
func TestCAIsARootThatSignsLeavesAlone(t *testing.T) {
	dir := aCA(t)
	data, err := os.ReadFile(filepath.Join(dir, "mitmproxy-ca.crt"))
	// Stop when the cert fails to read
	if err != nil {
		t.Fatalf("ReadFile() = %v", err)
	}
	block, _ := pem.Decode(data)

	cert, err := x509.ParseCertificate(block.Bytes)

	// Stop when the cert fails to parse
	if err != nil {
		t.Fatalf("ParseCertificate() = %v", err)
	}
	// Require a root CA with no CA below it
	if !cert.IsCA || cert.MaxPathLen != 0 || !cert.MaxPathLenZero {
		t.Errorf("cert IsCA %v, MaxPathLen %d, want a CA with path length 0", cert.IsCA, cert.MaxPathLen)
	}
	// Require the usages a signing root needs
	if cert.KeyUsage != x509.KeyUsageCertSign|x509.KeyUsageCRLSign {
		t.Errorf("cert KeyUsage = %v, want cert and CRL signing", cert.KeyUsage)
	}
	// Require the name the proxy's CA carries
	if got := cert.Subject.String(); got != "CN=Egress Proxy CA,O=o3s" {
		t.Errorf("cert subject = %q, want CN=Egress Proxy CA,O=o3s", got)
	}
}

// The proxy reads one file that holds the key first and the cert after it.
func TestCAPemHoldsTheKeyThenTheCert(t *testing.T) {
	dir := aCA(t)
	data, err := os.ReadFile(filepath.Join(dir, "mitmproxy-ca.pem"))
	// Stop when the file fails to read
	if err != nil {
		t.Fatalf("ReadFile() = %v", err)
	}

	key, rest := pem.Decode(data)
	cert, _ := pem.Decode(rest)

	// Require the key, then the cert
	if key == nil || cert == nil || key.Type != "PRIVATE KEY" || cert.Type != "CERTIFICATE" {
		t.Errorf("mitmproxy-ca.pem holds %v then %v, want a private key then a certificate", key, cert)
	}
}

// The DH parameters are the 2048-bit group of RFC 3526 with generator 2.
func TestCADhParamsAreTheRfc3526Group(t *testing.T) {
	dir := aCA(t)
	data, err := os.ReadFile(filepath.Join(dir, "mitmproxy-dhparam.pem"))
	// Stop when the file fails to read
	if err != nil {
		t.Fatalf("ReadFile() = %v", err)
	}
	block, _ := pem.Decode(data)
	var params struct{ P, G *big.Int }

	_, err = asn1.Unmarshal(block.Bytes, &params)

	// Stop when the parameters fail to parse
	if err != nil || block.Type != "DH PARAMETERS" {
		t.Fatalf("DH parameters %q: %v", block.Type, err)
	}
	want, _ := new(big.Int).SetString(rfc3526Group14, 16)
	// Require the RFC's prime and generator
	if params.P.Cmp(want) != 0 || params.G.Int64() != 2 {
		t.Errorf("DH parameters differ from RFC 3526 group 14 with g = 2")
	}
}

// A second run keeps the CA it made, so the cage's trust stays valid.
func TestCAKeepsWhatItMade(t *testing.T) {
	dir := aCA(t)
	before, _ := os.ReadFile(filepath.Join(dir, "mitmproxy-ca.pem"))

	made, err := CA(dir)

	after, _ := os.ReadFile(filepath.Join(dir, "mitmproxy-ca.pem"))
	// Require the CA kept and reported as such
	if err != nil || made || !bytes.Equal(before, after) {
		t.Errorf("CA() again = %v, %v, kept %v, want the CA kept", made, err, bytes.Equal(before, after))
	}
}
