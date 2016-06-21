package acmewrapper

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// savePrivateKey is used to write the given key to file
// This code copied from caddy:
// https://github.com/mholt/caddy/blob/master/caddy/https/crypto.go
func (w *AcmeWrapper) savePrivateKey(filename string, key crypto.PrivateKey) error {
	var pemType string
	var keyBytes []byte
	switch key := key.(type) {
	case *ecdsa.PrivateKey:
		var err error
		pemType = "EC"
		keyBytes, err = x509.MarshalECPrivateKey(key)
		if err != nil {
			return err
		}
	case *rsa.PrivateKey:
		pemType = "RSA"
		keyBytes = x509.MarshalPKCS1PrivateKey(key)
	}
	pemKey := pem.Block{Type: pemType + " PRIVATE KEY", Bytes: keyBytes}
	pemEncoded := bytes.Buffer{}
	if err := pem.Encode(&pemEncoded, &pemKey); err != nil {
		return err
	}
	return w.saveFile(filename, pemEncoded.Bytes())
}

// loadPrivateKey reads a key from file
// This code copied from caddy:
// https://github.com/mholt/caddy/blob/master/caddy/https/crypto.go
func (w *AcmeWrapper) loadPrivateKey(filename string) (crypto.PrivateKey, error) {
	keyBytes, err := w.loadFile(filename)
	if err != nil {
		return nil, err
	}
	keyBlock, _ := pem.Decode(keyBytes)
	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(keyBlock.Bytes)
	}
	return nil, errors.New("unknown private key type")
}

// AcmeWrapper version of LoadX509KeyPair reads and parses a public/private key pair from a pair of
// files. The files must contain PEM encoded data. Pulled from std lib tls.go.
func (w *AcmeWrapper) loadX509KeyPair(certFile, keyFile string) (tls.Certificate, error) {
	certPEMBlock, err := w.loadFile(certFile)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEMBlock, err := w.loadFile(keyFile)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(certPEMBlock, keyPEMBlock)
}
