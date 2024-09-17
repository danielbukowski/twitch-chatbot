package command

import (
	"fmt"

	"github.com/gempir/go-twitch-irc/v4"
)

var Ping CallbackSignature = func(_ []string, message *twitch.PrivateMessage, chatClient chatClient) error {
	chatClient.Say(message.Channel, fmt.Sprintf("Pong! @%s", message.User.DisplayName))

	return nil
}
