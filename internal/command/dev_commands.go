package command

import (
	"context"
	"fmt"
)

var Ping Handler = func(ctx context.Context, _ []string, chatClient chatClient) error {
	cmdCtx := UnwrapContext(ctx)
	chatClient.Say(cmdCtx.PrivMsg.Channel, fmt.Sprintf("Pong! @%s", cmdCtx.PrivMsg.User.DisplayName))

	return nil
}
