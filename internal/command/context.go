package command

import (
	"context"

	"github.com/gempir/go-twitch-irc/v4"
	"go.uber.org/zap"
)

// Context represents a struct with helping fields for validating and recording a command workflow.
type Context struct {
	CommandName string                 // CommandName represents a name of a current command.
	PrivMsg     *twitch.PrivateMessage // PrivMsg represents metadata of the sent message.
	Logger      *zap.Logger            // Logger records and captures events.
}

type contextKey struct{}

var key contextKey

// NewContext returns a new instance of a struct with metadata and a logger to a command lifecycle.
func NewContext(commandName string, privMsg *twitch.PrivateMessage, logger *zap.Logger) *Context {
	return &Context{
		CommandName: commandName,
		PrivMsg:     privMsg,
		Logger:      logger.Named("command"),
	}
}

// setContextToCommand binds Command Context to a context.
func setContextToCommand(ctx context.Context, commandContext *Context) context.Context {
	return context.WithValue(ctx, key, commandContext)
}

// UnwrapContext returns Command Context from a context.
func UnwrapContext(ctx context.Context) *Context {
	v, _ := ctx.Value(key).(*Context)
	return v
}
