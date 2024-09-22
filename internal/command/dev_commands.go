package command

import (
	"context"
	"fmt"
)

var Ping Handler = func(ctx context.Context, _ []string, chatClient chatClient) error {
	privMsg := GetPrivateMessageFromContext(ctx)
	chatClient.Say(privMsg.Channel, fmt.Sprintf("Pong! @%s", privMsg.User.DisplayName))

	return nil
}
