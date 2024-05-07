package lib

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// StringToBase64 converts a string to base64
func StringToBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// Base64ToString converts a base64 to string
func Base64ToString(b string) string {
	if s, err := base64.StdEncoding.DecodeString(b); err == nil {
		return string(s)
	}
	return ""
}

// SignHMAC256 returns the HMAC signature for a given message
func SignHMAC256(message, secret string) string {
	sig := hmac.New(sha256.New, []byte(secret))
	sig.Write([]byte(message))
	return hex.EncodeToString(sig.Sum(nil))
}

// CreateToken returns a token that can later be validated for tampering and expiry
func CreateToken(value, secret string, mins int) string {
	t := strconv.FormatInt(time.Now().UTC().Add(time.Duration(mins)*time.Minute).Unix(), 10)
	message := StringToBase64(value + "." + t)
	signature := SignHMAC256(message, secret)
	return message + "." + signature
}

// ValidateToken validates that a given token's value wansn't tampered with and that it didn't expire. It returns the value the token held.
func ValidateToken(token, secret string) (string, bool) {
	if token == "" {
		return "no token", false
	}
	parts := strings.Split(token, ".")
	signature := parts[1]
	if signature != SignHMAC256(parts[0], secret) {
		return "signature mismatch", false
	}
	parts = strings.Split(Base64ToString(parts[0]), ".")
	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "can't parse time", false
	}
	if time.Now().UTC().Unix() >= expiry {
		return "expired", false
	}
	return parts[0], true
}

func Encrypt(plaintext, key string) string {
	ciphertext, err := EncryptErr(plaintext, key)
	Check(err)
	return ciphertext
}

func EncryptErr(plaintext, key string) (string, error) {
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}
	c, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := string(gcm.Seal(nonce, nonce, []byte(plaintext), nil))
	return StringToBase64(ciphertext), nil
}

func Decrypt(ciphertext, key string) string {
	plaintext, err := DecryptErr(ciphertext, key)
	Check(err)
	return plaintext
}

func DecryptErr(ciphertext, key string) (string, error) {
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}
	c, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	ciphertext = Base64ToString(ciphertext)
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	out, err := gcm.Open(nil, []byte(nonce), []byte(ciphertext), nil)
	return string(out), err
}

func SecretsLoad(secret string, secrets map[string]string) {
	for k, v := range secrets {
		if strings.HasPrefix(v, "$e1$") {
			os.Setenv(k, SecretsDecrypt(secret, v[4:]))
		} else {
			os.Setenv(k, v)
		}
	}
}

func SecretsDecrypt(secret, cipherText string) string {
	key, err := hex.DecodeString(secret)
	Check(err)
	c, err := aes.NewCipher([]byte(key))
	Check(err)
	gcm, err := cipher.NewGCM(c)
	Check(err)
	cipherText = Base64ToString(cipherText)
	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		Check(errors.New("secretsDecrypt: cipher text smaller than nonce"))
	}
	nonce, encryptedText := cipherText[:nonceSize], cipherText[nonceSize:]
	text, err := gcm.Open(nil, []byte(nonce), []byte(encryptedText), nil)
	Check(err)
	return string(text)
}

func SecretsEncrypt(secret, text string) string {
	key, err := hex.DecodeString(secret)
	Check(err)
	c, err := aes.NewCipher(key)
	Check(err)
	gcm, err := cipher.NewGCM(c)
	Check(err)
	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	Check(err)
	return StringToBase64(string(gcm.Seal(nonce, nonce, []byte(text), nil)))
}
