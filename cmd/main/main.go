package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/danielbukowski/twitch-chatbot/internal/access_credentials/cipher"
	"github.com/danielbukowski/twitch-chatbot/internal/access_credentials/storage"
	"github.com/danielbukowski/twitch-chatbot/internal/command"
	"github.com/danielbukowski/twitch-chatbot/internal/config"
	"github.com/gempir/go-twitch-irc/v4"
	"github.com/nicklaw5/helix/v2"
)

func main() {
	ctx := context.Background()

	isDevEnv := flag.Bool("dev", false, "development environment check")
	code := flag.String("code", "", "twitch authorization code to get access credentials")
	flag.Parse()

	config, err := config.New(*isDevEnv)
	if err != nil {
		panic(err)
	}

	accessCredentialsCipher, err := cipher.NewAESCipher(config.CipherPassphrase, 24)
	if err != nil {
		panic(err)
	}

	accessCredentialsStorage, err := storage.NewSQLiteStorage("file:./db/database.db", config.DatabaseUsername, config.DatabasePassword, accessCredentialsCipher)
	if err != nil {
		panic(err)
	}

	helixClient, err := helix.NewClient(&helix.Options{
		ClientID:     config.TwitchClientID,
		ClientSecret: config.TwitchClientSecret,
		RedirectURI:  config.TwitchOAUTH2RedirectURI,
	})
	if err != nil {
		panic(err)
	}

	if *isDevEnv && len(*code) != 0 {
		fmt.Println("exchanging authorization code for access credentials...")

		resp, err := helixClient.RequestUserAccessToken(*code)
		if err != nil || resp.StatusCode != 200 {
			panic(errors.Join(errors.New("failed to exchange the code for access credentials"), err))
		}

		err = accessCredentialsStorage.Save(ctx, resp.Data, config.TwitchChannelName)
		if err != nil {
			panic(errors.Join(errors.New("failed to save the exchanged access credentials to database"), err))
		}

		fmt.Println("successfully exchanged and saved access credentials!")
	}

	accessCredentials, err := accessCredentialsStorage.Retrieve(ctx, config.TwitchChannelName)
	if err != nil {
		panic(err)
	}

	ircClient := twitch.NewClient(config.TwitchChatbotName, fmt.Sprintf("oauth:%s", accessCredentials.AccessToken))
	ircClient.Join(config.TwitchChannelName)

	commandController := command.NewController()
	commandPrefix := commandController.Prefix()

	if *isDevEnv {
		commandController.AddCommand(commandPrefix+"ping", command.Ping)
	}

	ircClient.OnPrivateMessage(func(privateMessage twitch.PrivateMessage) {
		userMessage := privateMessage.Message

		if !strings.HasPrefix(userMessage, commandPrefix) {
			return
		}

		commandController.CallCommand(userMessage, &privateMessage, ircClient)
	})

	ircClient.OnConnect(func() {
		fmt.Println("Connected to the chat!")
	})

	if isAccessTokenValid, _, err := helixClient.ValidateToken(accessCredentials.AccessToken); !isAccessTokenValid && err == nil {
		fmt.Println("access credentials are expired")

		resp, err := helixClient.RefreshUserAccessToken(accessCredentials.RefreshToken)
		if err != nil || resp.StatusCode != 200 {
			panic(errors.Join(errors.New("failed to refresh access credentials"), err))
		}

		fmt.Println("refreshed access credentials")

		err = accessCredentialsStorage.Update(ctx, resp.Data, config.TwitchChannelName)
		if err != nil {
			panic(err)
		}

		fmt.Println("saved new access credentials to database")

		ircClient.SetIRCToken(fmt.Sprintf("oauth:%s", resp.Data.AccessToken))
	}

	err = ircClient.Connect()
	if err != nil {
		panic(errors.Join(errors.New("failed to connect to the IRC server"), err))
	}
}
