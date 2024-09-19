package command

import (
	"errors"
	"strings"

	"github.com/gempir/go-twitch-irc/v4"
	"go.uber.org/zap"
)

type CallbackSignature func(args []string, privateMessage *twitch.PrivateMessage, chatClient chatClient) error

type Filter func(CallbackSignature) CallbackSignature

type chatClient interface {
	Say(channelName, message string)
	Reply(channelName, parentMessageID, message string)
	Join(channels ...string)
	Depart(channelName string)
}

type Controller struct {
	logger   *zap.Logger
	commands map[string]CallbackSignature
	prefix   string
}

func NewController(prefix string, logger *zap.Logger) *Controller {
	return &Controller{
		logger:   logger,
		commands: make(map[string]CallbackSignature),
		prefix:   prefix,
	}
}

func (c Controller) Prefix() string {
	return c.prefix
}

func (c *Controller) CallCommand(userMessage string, privateMessage *twitch.PrivateMessage, chatClient chatClient) {
	args := strings.Split(userMessage, " ")
	commandName := args[0]

	command, ok := c.commands[commandName]
	if !ok {
		return
	}

	err := command(args[1:], privateMessage, chatClient)

	if err != nil {
		if errors.Is(err, errNoPermissions) {
			return
		}

		c.logger.Error("An unhandled error occurred", zap.Error(err))
	}
}

func (c *Controller) AddCommand(commandName string, cb CallbackSignature, filters []Filter) {
	for i := len(filters) - 1; i >= 0; i-- {
		cb = filters[i](cb)
	}

	c.commands[commandName] = cb
}
