package config

import (
	"errors"
	"fmt"
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
	GrafanaCloudInstanceID  string
	GrafanaAPIToken         string
	OTELServiceName         string
	OTLPExporterEndpoint    string
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
		TwitchClientID:          getEnv("TWITCH_CLIENT_ID"),
		TwitchClientSecret:      getEnv("TWITCH_CLIENT_SECRET"),
		TwitchChatbotName:       getEnv("TWITCH_CHATBOT_NAME"),
		TwitchChannelName:       getEnv("TWITCH_CHANNEL_NAME"),
		TwitchOAuth2RedirectURI: getEnv("TWITCH_OAUTH2_REDIRECT_URI"),
		CipherPassphrase:        getEnv("CIPHER_PASSPHRASE"),
		DatabaseUsername:        getEnv("DATABASE_USERNAME"),
		DatabasePassword:        getEnv("DATABASE_PASSWORD"),
		GrafanaCloudInstanceID:  getEnv("GRAFANA_CLOUD_INSTANCE_ID"),
		GrafanaAPIToken:         getEnv("GRAFANA_API_TOKEN"),
		OTELServiceName:         getEnv("OTEL_SERVICE_NAME"),
		OTLPExporterEndpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}, nil
}

func getEnv(name string) string {
	env := os.Getenv(name)
	if len(env) == 0 {
		panic(fmt.Sprintf("environment variable called '%s' is missing", name))
	}

	return env
}
