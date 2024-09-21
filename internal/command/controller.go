package command

import (
	"context"
	"strings"

	"github.com/gempir/go-twitch-irc/v4"
	"go.uber.org/zap"
)

type CallbackSignature func(ctx context.Context, args []string, chatClient chatClient) error

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

func (c *Controller) CallCommand(ctx context.Context, userMessage string, privateMessage twitch.PrivateMessage, chatClient chatClient) {
	args := strings.Split(userMessage, " ")
	commandName := args[0]

	command, ok := c.commands[commandName]
	if !ok {
		return
	}

	ctx = setPrivateMessageToContext(ctx, &privateMessage)

	//nolint:errcheck // error is handled in a Middleware called ErrorHandler
	command(ctx, args[1:], chatClient)
}


func (c *Controller) AddCommand(commandName string, cb CallbackSignature, filters []Filter) {
	for i := len(filters) - 1; i >= 0; i-- {
		cb = filters[i](cb)
	}

	c.commands[commandName] = cb
}
