package cipher

import (
	"testing"

	"github.com/nicklaw5/helix/v2"
)

func TestEncryptionAndDecryptionFlow(t *testing.T) {

	// given
	accessCredentials := helix.AccessCredentials{
		AccessToken:  "iuwusa878o2jnkdsah1gdaljaaa232ss",
		RefreshToken: "clhjdsaiujnxztya3sdydaskbdsa2313",
		ExpiresIn:    2137,
		Scopes:       []string{"read:chat"},
	}
	passphrase := "supersecretpassword123lol"

	AESCipher, err := NewAESCipher(passphrase, 24)
	if err != nil {
		t.Fatalf("failed to initialize AES cipher: %v", err)
	}

	// when && then
	encryptedAccessCredentials, err := AESCipher.Encrypt(accessCredentials)
	if err != nil {
		t.Fatalf("failed to encrypt access credentials: %v", err)
	}

	_, err = AESCipher.Decrypt(encryptedAccessCredentials)
	if err != nil {
		t.Fatalf("failed to decrypt access credentials: %v", err)
	}

}
