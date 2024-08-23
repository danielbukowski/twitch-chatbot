package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/danielbukowski/twitch-chatbot/internal/command"
	credentialstorage "github.com/danielbukowski/twitch-chatbot/internal/credential_storage"
	"github.com/gempir/go-twitch-irc/v4"
	"github.com/joho/godotenv"
	"github.com/nicklaw5/helix/v2"
)

func main() {
	err := godotenv.Load("../../.dev.env")
	if err != nil {
		panic(err)
	}

	accessCredentials, err := credentialstorage.RetrieveAccessCredentialsFromFile()
	if err != nil {
		panic(err)
	}

	helixClient, err := helix.NewClient(&helix.Options{
		ClientID:     os.Getenv("TWITCH_CLIENT_ID"),
		ClientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
	})

	if err != nil {
		panic(err)
	}

	chatbotName := os.Getenv("CHATBOT_NAME")
	channelName := os.Getenv("CHANNEL_NAME")

	ircClient := twitch.NewClient(chatbotName, "oauth:"+accessCredentials.AccessToken)
	ircClient.Join(channelName)

	commandController := command.NewController()
	commandPrefix := commandController.Prefix()

	commandController.AddCommand(commandPrefix+"ping", command.Ping)

	ircClient.OnPrivateMessage(func(privateMessage twitch.PrivateMessage) {
		userMessage := privateMessage.Message

		if !strings.HasPrefix(userMessage, commandPrefix) {
			return
		}

		commandController.CallCommand(userMessage, &privateMessage, ircClient)
	})

	ircClient.OnConnect(func() {
		fmt.Println("Connected to the chat")
		ircClient.Say(channelName, "yo")
	})

	err = ircClient.Connect()
	if err != nil {
		fmt.Println("Failed to connect to the IRC server:", err.Error())

		if isAccessTokenValid, _, err := helixClient.ValidateToken(accessCredentials.AccessToken); !isAccessTokenValid && err == nil {
			fmt.Println("Trying to refresh the access token...")
			resp, err := helixClient.RefreshUserAccessToken(accessCredentials.RefreshToken)

			if err != nil || resp.StatusCode != 200 {
				fmt.Println("Failed to refresh the access token")
				os.Exit(1)
			}

			fmt.Println("Refreshed the access token!")

			err = credentialstorage.SaveAccessCredentialsToFile(resp.Data)
			if err != nil {
				fmt.Println("Failed to save the access token to a file")
				panic(err)
			}

			fmt.Println("Successfully saved the access token! Restart the application")
		}
	}
}
