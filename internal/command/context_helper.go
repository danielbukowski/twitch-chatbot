package command

import (
	"context"

	"github.com/gempir/go-twitch-irc/v4"
)

type privMsgKey struct{}

var key privMsgKey

func GetPrivateMessageFromContext(ctx context.Context) *twitch.PrivateMessage {
	privMsg, ok := ctx.Value(key).(*twitch.PrivateMessage)

	if !ok {
		panic("called GetPrivateMessageFromContext() function outside of scopes of Filters, Middlewares or Handlers")
	}

	return privMsg
}

func setPrivateMessageToContext(ctx context.Context, privateMessage *twitch.PrivateMessage) context.Context {
	return context.WithValue(ctx, key, privateMessage)
}
