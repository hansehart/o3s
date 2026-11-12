// Package host prepares the machine an o3s home runs on: the proxy's CA, the cage's confinement and the values compose reads.
package host

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/hansehart/o3s/internal/atomicfile"
)

// keyBits is the size of the CA's RSA key.
var keyBits = 4096

// group14 is the 2048-bit MODP prime of RFC 3526, the DH group the proxy offers.
const group14 = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF"

// CA makes the proxy's CA in a folder once, keeps what the folder holds, and reports whether it made a new one.
func CA(dir string) (bool, error) {
	// Hand back a folder that stays uncreated
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, err
	}
	// Keep the folder for its owner alone, as an earlier run may have left it otherwise
	if err := os.Chmod(dir, 0o700); err != nil { //nolint:gosec // a folder needs its search bit, and only its owner gets it
		return false, err
	}
	pemPath, crtPath, dhPath := filepath.Join(dir, "mitmproxy-ca.pem"), filepath.Join(dir, "mitmproxy-ca.crt"), filepath.Join(dir, "mitmproxy-dhparam.pem")
	made := false
	// Make the CA where the folder holds none
	if missing(pemPath) {
		key, cert, err := newCA()
		// Hand back a CA that fails to make
		if err != nil {
			return false, err
		}
		// Hand back a CA that stays unwritten
		if err := atomicfile.Write(pemPath, append(key, cert...), 0o600); err != nil {
			return false, err
		}
		// Hand back a public cert that stays unwritten
		if err := atomicfile.Write(crtPath, cert, 0o644); err != nil {
			return false, err
		}
		made = true
	}
	// Derive the public cert from the CA where it went missing
	if missing(crtPath) {
		cert, err := certOf(pemPath)
		// Hand back a CA whose cert fails to read
		if err != nil {
			return made, err
		}
		// Hand back a public cert that stays unwritten
		if err := atomicfile.Write(crtPath, cert, 0o644); err != nil {
			return made, err
		}
	}
	// Let the cage read the public cert, whatever mode it had
	if err := os.Chmod(crtPath, 0o644); err != nil { //nolint:gosec // the cage trusts this public cert, so every reader may read it
		return made, err
	}
	// Write the DH group where the folder holds none, so the proxy finds its config complete
	if missing(dhPath) {
		// Hand back DH parameters that stay unwritten
		if err := atomicfile.Write(dhPath, dhParams(), 0o600); err != nil {
			return made, err
		}
	}
	return made, nil
}

// newCA makes a self-signed root that signs leaf certs alone, as PEM blocks of its key and cert.
func newCA() ([]byte, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	// Hand back a key that fails to make
	if err != nil {
		return nil, nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	// Hand back a serial that fails to draw
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{Organization: []string{"o3s"}, CommonName: "Egress Proxy CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	// Hand back a cert that fails to sign
	if err != nil {
		return nil, nil, err
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	// Hand back a key that fails to encode
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), nil
}

// certOf reads the cert block from the file that holds the CA.
func certOf(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	// Hand back a CA that fails to read
	if err != nil {
		return nil, err
	}
	// Look at every block for the cert
	for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
		// Take the cert block
		if block.Type == "CERTIFICATE" {
			return pem.EncodeToMemory(block), nil
		}
	}
	return nil, fmt.Errorf("%s holds no certificate", path)
}

// dhParams encodes the RFC 3526 group with generator 2 as the PEM the proxy loads.
func dhParams() []byte {
	p, _ := new(big.Int).SetString(group14, 16)
	der, _ := asn1.Marshal(struct{ P, G *big.Int }{p, big.NewInt(2)})
	return pem.EncodeToMemory(&pem.Block{Type: "DH PARAMETERS", Bytes: der})
}

// missing reports whether a file is absent.
func missing(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, fs.ErrNotExist)
}
