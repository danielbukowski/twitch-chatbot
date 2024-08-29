package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gempir/go-twitch-irc/v4"
)

type CallbackSignature func(args []string, privateMessage *twitch.PrivateMessage, ircClient *twitch.Client) error

type Controller struct {
	commands map[string]CallbackSignature
	prefix   string
}

func NewController() *Controller {
	return &Controller{
		commands: make(map[string]CallbackSignature),
		prefix:   "!",
	}
}

func (c Controller) Prefix() string {
	return c.prefix
}

func (c *Controller) CallCommand(userMessage string, privateMessage *twitch.PrivateMessage, ircClient *twitch.Client) {
	args := strings.Split(userMessage, " ")
	commandName := args[0]

	command := c.commands[commandName]
	if command == nil {
		return
	}

	err := command(args[1:], privateMessage, ircClient)

	if err != nil {
		if errors.Is(err, errNoPermissions) {
			return
		}

		fmt.Println("An unhandled error occurred: ", err.Error())
	}
}

func (c *Controller) AddCommand(commandName string, cb CallbackSignature) {
	c.commands[commandName] = cb
}
