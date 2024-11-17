package command

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

func ErrorHandler() Middleware {
	return func(cb Handler) Handler {
		return func(ctx context.Context, args []string, chatClient chatClient) error {
			spanCtx, span := tracer.Start(ctx, "errorHandler")
			defer span.End()

			cmdCtx := UnwrapContext(ctx)
			span.SetAttributes(attribute.String("command.name", cmdCtx.CommandName))
			span.SetAttributes(attribute.String("command.caller.name", cmdCtx.PrivMsg.User.DisplayName))

			err := cb(spanCtx, args, chatClient)

			if err != nil {
				if errors.Is(err, errNoPermissions) {
					span.SetStatus(codes.Error, "got error NoPermissions from a command")
					return nil
				}
				if errors.Is(err, errCommandOnCooldown) {
					span.SetStatus(codes.Error, "got error CommandOnCooldown from a command")
					return nil
				}

				span.SetStatus(codes.Error, "unhandled error occurred")
				cmdCtx.Logger.Error("unhandled error occurred", zap.Error(err))
				return nil
			}
			span.SetStatus(codes.Ok, "successfully executed a command without error")
			return nil
		}
	}
}
