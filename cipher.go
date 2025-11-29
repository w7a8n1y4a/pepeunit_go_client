package pepeunit

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

type aesGcmCipher struct{}

func (a *aesGcmCipher) encode(data string, keyB64 string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return "", err
	}
	if l := len(key); l != 16 && l != 24 && l != 32 {
		return "", errors.New("invalid AES key length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(data), nil)
	return base64.StdEncoding.EncodeToString(nonce) + "." + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (a *aesGcmCipher) decode(encoded string, keyB64 string) (string, error) {
	parts := strings.Split(encoded, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid encoded data format")
	}
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return "", err
	}
	if l := len(key); l != 16 && l != 24 && l != 32 {
		return "", errors.New("invalid AES key length")
	}
	nonce, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (c *PepeunitClient) AESGCMEncode(data string, keyB64 string) (string, error) {
	a := &aesGcmCipher{}
	return a.encode(data, keyB64)
}

func (c *PepeunitClient) AESGCMDecode(encoded string, keyB64 string) (string, error) {
	a := &aesGcmCipher{}
	return a.decode(encoded, keyB64)
}
