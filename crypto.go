package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

var (
	ErrInvalidData   = errors.New("invalid data")
	ErrDecryptFailed = errors.New("decryption failed")
	ErrIntegrity     = errors.New("integrity check failed")
)

// DeriveKey creates key from password using Argon2id
func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		Argon2Time,
		Argon2Memory,
		Argon2Threads,
		Argon2KeyLength,
	)
}

// DeriveKeyFast for session verification (not for encryption)
func DeriveKeyFast(password string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		1,
		64*1024,
		4,
		Argon2KeyLength,
	)
}

// GenerateSalt creates random salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	_, err := io.ReadFull(rand.Reader, salt)
	return salt, err
}

// GenerateNonce creates random nonce
func GenerateNonce(size int) ([]byte, error) {
	nonce := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, nonce)
	return nonce, err
}

// EncryptAESGCM encrypts data with AES-256-GCM
func EncryptAESGCM(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, err := GenerateNonce(gcm.NonceSize())
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result, nonce)
	copy(result[len(nonce):], ciphertext)

	return result, nil
}

// DecryptAESGCM decrypts AES-256-GCM data
func DecryptAESGCM(encrypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(encrypted) < gcm.NonceSize() {
		return nil, ErrInvalidData
	}

	nonce := encrypted[:gcm.NonceSize()]
	ciphertext := encrypted[gcm.NonceSize():]

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// EncryptXChaCha20 encrypts with XChaCha20-Poly1305
func EncryptXChaCha20(plaintext, key []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	nonce, err := GenerateNonce(aead.NonceSize())
	if err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result, nonce)
	copy(result[len(nonce):], ciphertext)

	return result, nil
}

// DecryptXChaCha20 decrypts XChaCha20-Poly1305 data
func DecryptXChaCha20(encrypted, key []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	if len(encrypted) < aead.NonceSize() {
		return nil, ErrInvalidData
	}

	nonce := encrypted[:aead.NonceSize()]
	ciphertext := encrypted[aead.NonceSize():]

	return aead.Open(nil, nonce, ciphertext, nil)
}

// Encrypt encrypts with password (full process)
// Format: flag(1) + salt(32) + encrypted_data
// flag: 0x01 = single AES, 0x02 = double (AES+XChaCha)
func Encrypt(plaintext []byte, password string) ([]byte, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	key := DeriveKey(password, salt)

	var encrypted []byte
	var flag byte = 0x01 // single encryption

	if AppConfig.DoubleEncrypt {
		flag = 0x02 // double encryption

		// Layer 1: AES-256-GCM
		layer1, err := EncryptAESGCM(plaintext, key)
		if err != nil {
			return nil, err
		}

		// Derive second key
		key2 := DeriveSecondKey(key)

		// Layer 2: XChaCha20-Poly1305
		encrypted, err = EncryptXChaCha20(layer1, key2)
		if err != nil {
			return nil, err
		}
	} else {
		encrypted, err = EncryptAESGCM(plaintext, key)
		if err != nil {
			return nil, err
		}
	}

	// Format: flag(1) + salt(32) + encrypted
	result := make([]byte, 1+SaltSize+len(encrypted))
	result[0] = flag
	copy(result[1:], salt)
	copy(result[1+SaltSize:], encrypted)

	return result, nil
}

// Decrypt decrypts with password
// Reads encryption mode from data itself
func Decrypt(encrypted []byte, password string) ([]byte, error) {
	if len(encrypted) < 1+SaltSize {
		return nil, ErrInvalidData
	}

	flag := encrypted[0]
	salt := encrypted[1 : 1+SaltSize]
	data := encrypted[1+SaltSize:]

	key := DeriveKey(password, salt)

	var plaintext []byte
	var err error

	if flag == 0x02 {
		// Double encryption
		key2 := DeriveSecondKey(key)

		// Layer 2: XChaCha20
		layer1, err := DecryptXChaCha20(data, key2)
		if err != nil {
			return nil, ErrDecryptFailed
		}

		// Layer 1: AES-GCM
		plaintext, err = DecryptAESGCM(layer1, key)
		if err != nil {
			return nil, ErrDecryptFailed
		}
	} else {
		// Single encryption (flag == 0x01 or old format)
		plaintext, err = DecryptAESGCM(data, key)
		if err != nil {
			return nil, ErrDecryptFailed
		}
	}

	return plaintext, nil
}

// DeriveSecondKey derives second key for double encryption
func DeriveSecondKey(key []byte) []byte {
	h := sha512.Sum512(key)
	return h[:32]
}

// HashPassword creates password hash for verification
func HashPassword(password string, salt []byte) []byte {
	key := DeriveKeyFast(password, salt)
	h := sha256.Sum256(key)
	return h[:]
}

// HMAC256 creates HMAC-SHA256
func HMAC256(data, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// VerifyHMAC verifies HMAC
func VerifyHMAC(data, mac, key []byte) bool {
	expected := HMAC256(data, key)
	return hmac.Equal(mac, expected)
}

// EncryptWithIntegrity adds HMAC for integrity
func EncryptWithIntegrity(plaintext []byte, password string) ([]byte, error) {
	encrypted, err := Encrypt(plaintext, password)
	if err != nil {
		return nil, err
	}

	// Add length prefix
	result := make([]byte, 4+len(encrypted)+32)
	binary.BigEndian.PutUint32(result[:4], uint32(len(encrypted)))
	copy(result[4:], encrypted)

	// HMAC of everything
	mac := HMAC256(result[:4+len(encrypted)], []byte(password))
	copy(result[4+len(encrypted):], mac)

	return result, nil
}

// DecryptWithIntegrity verifies and decrypts
func DecryptWithIntegrity(data []byte, password string) ([]byte, error) {
	if len(data) < 4+32 {
		return nil, ErrInvalidData
	}

	length := binary.BigEndian.Uint32(data[:4])
	if int(length) > len(data)-36 {
		return nil, ErrInvalidData
	}

	encrypted := data[4 : 4+length]
	mac := data[4+length : 4+length+32]

	// Verify HMAC
	if !VerifyHMAC(data[:4+length], mac, []byte(password)) {
		return nil, ErrIntegrity
	}

	return Decrypt(encrypted, password)
}

// SecureZero zeroes memory
func SecureZero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
