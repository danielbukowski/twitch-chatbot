package command

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/codes"
)

var Ping Handler = func(ctx context.Context, _ []string, chatClient chatClient) error {
	_, span := tracer.Start(ctx, "ping")
	defer span.End()

	cmdCtx := UnwrapContext(ctx)
	chatClient.Say(cmdCtx.PrivMsg.Channel, fmt.Sprintf("Pong! @%s", cmdCtx.PrivMsg.User.DisplayName))

	span.SetStatus(codes.Ok, "successfully sent a ping")
	return nil
}
