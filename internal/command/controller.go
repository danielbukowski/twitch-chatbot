package command

import (
	"context"
	"strings"

	"github.com/gempir/go-twitch-irc/v4"
	"go.uber.org/zap"
)

// Handler represents a function for a command.
type Handler func(ctx context.Context, args []string, chatClient chatClient) error

// Filter represents a function that is called after all middlewares and before a command. It is used for validation.
type Filter func(Handler) Handler

// Middleware represents a function that is called before any filters and a handler.
// It can be used for logging handler's parameters, error handling or measure how much time a command took to execute.
type Middleware func(Handler) Handler

// ChatClient provides an interface to interact with Twitch IRC.
type chatClient interface {
	Say(channelName, message string)                    // Say is function for sending messages to a twitch chat.
	Reply(channelName, parentMessageID, message string) // Reply is a function that replies to a thread.
	Join(channels ...string)                            // Join is a function that allows to join to a channel or channels.
	Depart(channelName string)                          // Depart is a function that allows to leave from a channel.
}

// Controller represents a manager to commands.
type Controller struct {
	logger      *zap.Logger        // Logger is just self explanatory, it's used for logging.
	commands    map[string]Handler // Commands is a map that stores callbacks for commands. Handlers are wrapped with middlewares and filters.
	middlewares []Middleware       // Middlewares represents a list of functions. Middlewares are added to every handlers before any filter.
	prefix      string             // Prefix represents a string that is added at the start of all keys in commands.
}

// NewController creates an instance of Controller for managing commands.
func NewController(prefix string, logger *zap.Logger) *Controller {
	return &Controller{
		logger:   logger,
		commands: make(map[string]Handler),
		prefix:   prefix,
	}
}

// CallCommand searches for a command and execute it, if find one.
func (c *Controller) CallCommand(ctx context.Context, userMessage string, privateMessage twitch.PrivateMessage, chatClient chatClient) {
	args := strings.Split(userMessage, " ")
	commandName := args[0]

	command, ok := c.commands[commandName]
	if !ok {
		return
	}

	ctx = setPrivateMessageToContext(ctx, &privateMessage)

	//nolint:errcheck // error is handled in a middleware called ErrorHandler
	command(ctx, args[1:], chatClient)
}

// UseWith adds a middleware to a middlewares. The order when a middleware is added matters.
func (c *Controller) UseWith(middleware Middleware) {
	c.middlewares = append(c.middlewares, middleware)
}

// AddCommand adds a command handler to a map in Controller.
// The handler is being wrapped with filters and middlewares, before it is added to commands.
// The order of functions in wrapped handler goes like this: Middlewares -> Filters -> Handler.
func (c *Controller) AddCommand(commandName string, handler Handler, filters []Filter) {
	for i := len(filters) - 1; i >= 0; i-- {
		handler = filters[i](handler)
	}

	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}

	c.commands[commandName] = handler
}
