package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

var privKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEArIHAC8hm3zqFbYS31vOpbXztYWngL1A1j0ArL7vl8gPSHOc/
qrdDM55HKQudCb1t9Ehvbj0AYBggZqdEscXKVeQ6sflblBDNmlkaUe3552vyrlxo
cuoUOsrhGhRULSeRpEPBUVl1ujO19g4K7mTISGJz706cugi7YA204AHhZjdrHkjc
nE0dPeAM8k67KmtStWAdN1eENcYv6P4tJC2aVWBGt4/vXvT80SuPoG0nLBHDuhX4
uUKyT7j4v+FVhtXPmIctdOU9HlqKIetIXG2x3c/F33D05gO/zldOWetkyB4UXdYh
7T8bk1EzYQUsiXC3c1RLYWe/hjaOycV92bWF9QIDAQABAoIBABJnhWW8a71FMRcO
Ic79EAZuXaUW9qQVmqE9512gXlgDsFcFFPXRt4m6ST1Hz+lKHIEDeV/5giC+cjaB
ggK1Z3DieKCkSfDyGf1tpDE5gn/+NFYF5d7cumvLWVi+8nwezJsJ34YeVUEYY0Ft
Hh1xAy0108ikSnROmi8VR0be7H1LH3p4xh89O5MnggxZY23ulifQklaepE/sW5Rm
xZ8hTh5gc0ZWOUKtzNLCY7E/NZ64mzLZxeffy666JJ7W2juUy9Nvcio7z6OI8cYJ
wxW05evxXIbGiqvPcVH8U9fpPYrmyMWGnu2CZQLJQ/f+CdRSKgp+psXtTeKGs6mV
j7nAngECgYEA1KKt5oQBvPjepElfS8a7dbaK6Uw1WpzAbV216J7fe5luPviyO/Yb
4cYlZBmCG64zxNLkZn11ZNa7tSazBHOpEzkSrzh8Q08CkxEXNVUZITfvEbb2h+O2
vWczOkcMHpifMtPP/Nj8uV8cHUM+yiD84ccffJl9FUrl08bJzhXEfw0CgYEAz7AG
uqJsXxVxG8b6ktsebqDbRma0Nx+8HLs7mT9XAgp/JouNdZLEntSSh13OFrKnvN9l
Y07szmfJlwpokNbfIyypmNQ1phAN9Vh2AqU1F1qgikR8IVpjYBSgpJ07DAnYYDWw
dAM46xgmx5JHMkj3HdmU1aSeXqsJnuHXBn2hqIkCgYEAuh/kBMFMHDPj/v5Ssiu0
72bn1SrmEPjg31qF3qcijDhTbj+XFDFrWodtN1+0qbIkkobfcm8jES99gyXq0VzE
hJUSq9nPu8MvR7hQDlj89DzXwTPcPDqxPL1W9A44p5eE0qLhjjwJi+Ga5Gp2Tcvu
T1tTTme0O60h2+xHvWMEawkCgYBp3gHgGzGA2o44qZpLR9TgcbC011+gBdY/ExYT
TR1OFgmwnUxzlWgSyqsSw7dtDI7CkWG+WJoPsQoqqwyYYF+IL5kYJXxfdSmYWgp1
2L8Sc4kIUlawLcIgVQN6Rge2YEJi8JtM1+OouzvxWVhfgqAtIB9lNC/ic+9IhTi1
v06reQKBgFt8SvcEg5js3YieO1/80ED07nE5wFLk1l6HoYCcmQhF4cVgqiQAsP0B
a9YQ5dZZTC55A2p5/X1DLKdlkllF8hOoxYNw0muTlR8te2e+HUVblRDioMu0fZKC
H6EyrO/Wg2zw5eJx/ZVQpB8lW4ggFiPh9FSmwQPDDKvFo18kVrP6
-----END RSA PRIVATE KEY-----`)

func BytesToPrivateKey(priv []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(priv)

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func DecryptWithPrivateKey(ciphertext []byte, priv *rsa.PrivateKey) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
}

func decrypt(ct string) (string, error) {
	// Decode the base64 ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(ct)
	if err != nil {
		fmt.Println("Decryption - Base64 error:", err)
		return "", err
	}

	// Decode the private key
	key, err := BytesToPrivateKey(privKey)
	if err != nil {
		fmt.Println("Decryption - BytesToPrivateKey error:", err)
		return "", err
	}

	// Decrypt the ciphertext
	plaintext, err := DecryptWithPrivateKey(ciphertext, key)
	if err != nil {
		fmt.Println("Decryption - DecryptWithPrivateKey error:", err)
		return "", err
	}

	return string(plaintext), nil
}
