package command

import (
	"fmt"

	"github.com/gempir/go-twitch-irc/v4"
)

var Ping CallbackSignature = func(args []string, message *twitch.PrivateMessage, ircClient *twitch.Client) error {
	ircClient.Say(message.Channel, fmt.Sprintf("Pong! @%s", message.User.DisplayName))

	return nil
}
