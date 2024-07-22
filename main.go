package main

import (
	"fmt"
	"os"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".dev.env")
	if err != nil {
		fmt.Println("Failed to load .dev.env file")
		return
	}

	accessToken := os.Getenv("ACCESS_TOKEN")
	chatbotName := os.Getenv("CHATBOT_NAME")
	channelName := os.Getenv("CHANNEL_NAME")

	client := twitch.NewClient(chatbotName, "oauth:"+accessToken)

	client.Join(channelName)

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		fmt.Printf("Message from %s: %s\n", message.User.DisplayName, message.Message)
	})

	client.OnConnect(func() {
		fmt.Println("Connected to the chat")
		client.Say(channelName, "peepoBlushCap")
	})

	err = client.Connect()
	if err != nil {
		fmt.Println("Failed to connect to the IRC server", err.Error())
	}
}
