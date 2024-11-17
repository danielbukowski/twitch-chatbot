package command

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/codes"
)

var errNoPermissions = errors.New("called a command without a needed role")
var errCommandOnCooldown = errors.New("command has a cooldown")

// From the twitch docs I found, that are available badges like:
// ["broadcaster", "moderator", "subscriber", "artist-badge", "founder", "vip", "sub-gifter", "bits", "partner", "staff"].
func hasBadge(badgeName string, badges map[string]int) bool {
	return badges[badgeName] != 0
}

// HasRole rejects user's command request, when the user does not have a role for that.
// The roles are compared with users's twitch badges.
func HasRole(roles []string) Filter {
	return func(cb Handler) Handler {
		return func(ctx context.Context, args []string, chatClient chatClient) error {
			spanCtx, span := tracer.Start(ctx, "hasRole")
			defer span.End()

			cmdCtx := UnwrapContext(ctx)

			for _, roleName := range roles {
				if hasBadge(roleName, cmdCtx.PrivMsg.User.Badges) {
					err := cb(spanCtx, args, chatClient)
					span.SetStatus(codes.Ok, "user passed through the filter")
					return err
				}
			}

			span.SetStatus(codes.Error, "user called a command without the required role")
			return errNoPermissions
		}
	}
}

// Cooldown stops from calling a command, when not enough time passed.
func Cooldown(cooldown time.Duration) Filter {
	lastCalled := time.Time{}

	return func(cb Handler) Handler {
		return func(ctx context.Context, args []string, chatClient chatClient) error {
			spanCtx, span := tracer.Start(ctx, "cooldown")
			defer span.End()

			if time.Now().Before(lastCalled.Add(cooldown)) {
				span.SetStatus(codes.Error, "not enough time passed to call a command")
				return errCommandOnCooldown
			}

			lastCalled = time.Now()
			span.SetStatus(codes.Ok, "user passed through the filter")
			err := cb(spanCtx, args, chatClient)
			return err
		}
	}
}
