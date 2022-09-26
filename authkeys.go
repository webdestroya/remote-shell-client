package main

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"

	"golang.org/x/crypto/ssh"
)

func generateSSHKeypair(bitSize int) (ssh.Signer, string, error) {
	privKey, err := generatePrivateKey(bitSize)
	if err != nil {
		return nil, "", err
	}

	signer, err := ssh.NewSignerFromKey(privKey)
	if err != nil {
		return nil, "", err
	}

	pubKey, err := generatePublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, "", err
	}

	return signer, pubKey, nil

}

func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func generatePublicKey(pubKey *rsa.PublicKey) (string, error) {
	publicRsaKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	pubKeyString := strings.TrimSuffix(string(pubKeyBytes), "\n")

	return pubKeyString, nil
}
