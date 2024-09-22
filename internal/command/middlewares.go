package command

import (
	"context"
	"errors"

	"go.uber.org/zap"
)

func ErrorHandler(logger *zap.Logger) Middleware {
	return func(cb Handler) Handler {
		return func(ctx context.Context, args []string, chatClient chatClient) error {

			err := cb(ctx, args, chatClient)

			if err != nil {
				if errors.Is(err, errNoPermissions) {
					return nil
				}

				logger.Error("An unhandled error occurred", zap.Error(err))
			}
			return nil
		}
	}
}
