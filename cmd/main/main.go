package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/danielbukowski/twitch-chatbot/internal/access_credentials/cipher"
	"github.com/danielbukowski/twitch-chatbot/internal/access_credentials/storage"
	"github.com/danielbukowski/twitch-chatbot/internal/command"
	"github.com/danielbukowski/twitch-chatbot/internal/config"
	"github.com/danielbukowski/twitch-chatbot/internal/logger"
	"github.com/gempir/go-twitch-irc/v4"
	"github.com/nicklaw5/helix/v2"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	isDevEnv := flag.Bool("dev", false, "development environment check")
	code := flag.String("code", "", "twitch authorization code to get access credentials")
	flag.Parse()

	logger, err := logger.New(*isDevEnv)
	if err != nil {
		panic(err)
	}

	//nolint:errcheck
	defer logger.Sync()

	logger.Info("successfully initialized logger", zap.Bool("IsDev", *isDevEnv))

	config, err := config.New(*isDevEnv)
	if err != nil {
		logger.Panic("failed to initialize config", zap.Error(err))
	}

	accessCredentialsCipher, err := cipher.NewAESCipher(config.CipherPassphrase, 24)
	if err != nil {
		logger.Panic("failed to create AES cipher", zap.Error(err))
	}

	accessCredentialsStorage, err := storage.NewSQLiteStorage(ctx, "file:./db/database.db", config.DatabaseUsername, config.DatabasePassword, accessCredentialsCipher, logger)
	if err != nil {
		logger.Panic("failed to establish a connection to SQLite", zap.Error(err))
	}

	helixClient, err := helix.NewClient(&helix.Options{
		ClientID:     config.TwitchClientID,
		ClientSecret: config.TwitchClientSecret,
		RedirectURI:  config.TwitchOAuth2RedirectURI,
	})
	if err != nil {
		panic(err)
	}

	if *isDevEnv && len(*code) != 0 {
		logger.Info("exchanging authorization code for access credentials...")

		resp, err := helixClient.RequestUserAccessToken(*code)
		if err != nil || resp.StatusCode != 200 {
			logger.Panic("failed to exchange the code for access credentials", zap.Error(err))
		}

		err = accessCredentialsStorage.Save(ctx, resp.Data, config.TwitchChannelName)
		if err != nil {
			logger.Panic("failed to save the exchanged access credentials to database", zap.Error(err))
		}

		logger.Info("successfully exchanged and saved access credentials!")
	}

	accessCredentials, err := accessCredentialsStorage.Retrieve(ctx, config.TwitchChannelName)
	if err != nil {
		logger.Panic("failed to retrieve access credentials from database", zap.Error(err))
	}

	ircClient := twitch.NewClient(config.TwitchChatbotName, fmt.Sprintf("oauth:%s", accessCredentials.AccessToken))
	ircClient.Join(config.TwitchChannelName)

	commandPrefix := "!"
	commandController := command.NewController(commandPrefix, logger)

	commandController.UseWith(command.ErrorHandler(logger))

	if *isDevEnv {

		// Add commands only after this line
		commandController.AddCommand(commandPrefix+"ping", command.Ping, []command.Filter{})
	}

	ircClient.OnPrivateMessage(func(privateMessage twitch.PrivateMessage) {
		userMessage := privateMessage.Message

		if !strings.HasPrefix(userMessage, commandPrefix) {
			return
		}

		commandController.CallCommand(ctx, userMessage, privateMessage, ircClient)
	})

	ircClient.OnConnect(func() {
		logger.Info("connected to the twitch chat!")
	})

	if isAccessTokenValid, _, err := helixClient.ValidateToken(accessCredentials.AccessToken); !isAccessTokenValid && err == nil {
		logger.Info("access credentials have expired")

		resp, err := helixClient.RefreshUserAccessToken(accessCredentials.RefreshToken)
		if err != nil || resp.StatusCode != 200 {
			logger.Panic("failed to refresh access credentials", zap.Error(err))
		}

		logger.Info("refreshed access credentials")

		err = accessCredentialsStorage.Update(ctx, resp.Data, config.TwitchChannelName)
		if err != nil {
			logger.Panic("failed to update access credentials", zap.Error(err))
		}

		logger.Info("saved new access credentials to database")

		ircClient.SetIRCToken(fmt.Sprintf("oauth:%s", resp.Data.AccessToken))
	}

	err = ircClient.Connect()
	if err != nil {
		logger.Panic("failed to connect to the IRC server", zap.Error(err))
	}
}
