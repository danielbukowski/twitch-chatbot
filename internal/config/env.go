package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TwitchClientID          string
	TwitchClientSecret      string
	TwitchChatbotName       string
	TwitchChannelName       string
	CipherPassphrase        string
	TwitchOAuth2RedirectURI string
	DatabaseUsername        string
	DatabasePassword        string
}

func New(isDevEnv bool) (*Config, error) {
	envFileName := ".env"

	if isDevEnv {
		envFileName = ".dev.env"
	}

	err := godotenv.Load(envFileName)
	if err != nil {
		return nil, errors.Join(errors.New("failed to load environment variables from a file"), err)
	}

	return &Config{
		TwitchClientID:          os.Getenv("TWITCH_CLIENT_ID"),
		TwitchClientSecret:      os.Getenv("TWITCH_CLIENT_SECRET"),
		TwitchChatbotName:       os.Getenv("TWITCH_CHATBOT_NAME"),
		TwitchChannelName:       os.Getenv("TWITCH_CHANNEL_NAME"),
		TwitchOAuth2RedirectURI: os.Getenv("TWITCH_OAUTH2_REDIRECT_URI"),
		CipherPassphrase:        os.Getenv("CIPHER_PASSPHRASE"),
		DatabaseUsername:        os.Getenv("DATABASE_USERNAME"),
		DatabasePassword:        os.Getenv("DATABASE_PASSWORD"),
	}, nil
}
