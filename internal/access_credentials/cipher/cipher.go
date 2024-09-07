package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/nicklaw5/helix/v2"
	"golang.org/x/crypto/pbkdf2"
)

type AESCipher struct {
	passphrase string
	keySize    int
}

const _PBKDF2SaltSize int = 16
const _PBKDF2Iterations int = 41732

func NewAESCipher(passphrase string, keySize int) (*AESCipher, error) {
	switch keySize {
	case 16, 24, 32:
		return &AESCipher{
			passphrase: passphrase,
			keySize:    keySize,
		}, nil
	default:
		return nil, errors.New("keySize should be 16, 24 or 32 bytes long")
	}
}

func (ac AESCipher) Encrypt(accessCredentials helix.AccessCredentials) (string, error) {
	salt := make([]byte, _PBKDF2SaltSize)

	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	key := generatePBKDF2Key(ac.passphrase, salt, ac.keySize)

	jsonString, err := json.Marshal(accessCredentials)
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

	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, jsonString, nil)

	saltNonceCiphertext := make([]byte, 0)
	saltNonceCiphertext = append(saltNonceCiphertext, salt...)
	saltNonceCiphertext = append(saltNonceCiphertext, nonce...)
	saltNonceCiphertext = append(saltNonceCiphertext, ciphertext...)

	return base64.StdEncoding.EncodeToString(saltNonceCiphertext), nil
}

func (ac AESCipher) Decrypt(base64SaltNonceCiphertext string) (helix.AccessCredentials, error) {
	saltNonceCiphertext, err := base64.StdEncoding.DecodeString(base64SaltNonceCiphertext)
	if err != nil {
		return helix.AccessCredentials{}, err
	}

	salt, nonceCiphertext := saltNonceCiphertext[:_PBKDF2SaltSize], saltNonceCiphertext[_PBKDF2SaltSize:]

	key := generatePBKDF2Key(ac.passphrase, salt, ac.keySize)

	block, err := aes.NewCipher(key)
	if err != nil {
		return helix.AccessCredentials{}, err
	}

	cipher, err := cipher.NewGCM(block)
	if err != nil {
		return helix.AccessCredentials{}, err
	}

	nonceSize := cipher.NonceSize()

	nonce := nonceCiphertext[:nonceSize]
	ciphertext := nonceCiphertext[nonceSize:]

	jsonString, err := cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return helix.AccessCredentials{}, err
	}

	var accessCredentials helix.AccessCredentials

	err = json.Unmarshal(jsonString, &accessCredentials)
	if err != nil {
		return helix.AccessCredentials{}, err
	}

	return accessCredentials, nil
}

func generatePBKDF2Key(passphrase string, salt []byte, keySize int) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, _PBKDF2Iterations, keySize, sha256.New)
}
