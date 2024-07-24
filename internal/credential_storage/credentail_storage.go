package tokenfetcher

import (
	"encoding/json"
	"os"

	"github.com/nicklaw5/helix/v2"
)

func RetrieveAccessCredentialsFromFile() (helix.AccessCredentials, error) {
	credentialsFile, err := os.ReadFile("../../access_credentials.json")
	if err != nil {
		return helix.AccessCredentials{}, err
	}

	var accessCredentials helix.AccessCredentials

	err = json.Unmarshal(credentialsFile, &accessCredentials)
	if err != nil {
		return helix.AccessCredentials{}, err
	}

	return accessCredentials, nil
}
