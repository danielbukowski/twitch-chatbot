package main

import (
	"fmt"
	"os"

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

	chatbotName := os.Getenv("CHATBOT_NAME")
	channelName := os.Getenv("CHANNEL_NAME")

	ircClient := twitch.NewClient(chatbotName, "oauth:"+accessCredentials.AccessToken)
	ircClient.Join(channelName)

	ircClient.OnPrivateMessage(func(message twitch.PrivateMessage) {
		fmt.Printf("Message from %s: %s\n", message.User.DisplayName, message.Message)
	})

	ircClient.OnConnect(func() {
		fmt.Println("Connected to the chat")
		ircClient.Say(channelName, "yo")
	})

	err = ircClient.Connect()
	if err != nil {
		fmt.Println("Failed to connect to the IRC server", err.Error())
	}
}
