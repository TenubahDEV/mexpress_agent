package updater

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

func VerifySignature(binaryPath, sigPath string) error {
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return err
	}

	sig, err := os.ReadFile(sigPath)
	if err != nil {
		return err
	}

	block, _ := pem.Decode([]byte(PublicKeyPEM))
	if block == nil {
		return errors.New("invalid public key")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(data)

	return rsa.VerifyPKCS1v15(
		pubKey.(*rsa.PublicKey),
		crypto.SHA256,
		hash[:],
		sig,
	)
}
