package command

import (
	"context"
	"errors"
	"time"
)

var errNoPermissions = errors.New("called a command without a needed role")
var errCommandOnCooldown = errors.New("command has a cooldown")

// Available twitch badges:
// ["broadcaster", "moderator", "subscriber", "artist-badge", "founder", "vip", "sub-gifter", "bits", "partner", "staff"].
func hasBadge(badgeName string, badges map[string]int) bool {
	return badges[badgeName] != 0
}

func HasRole(roles []string) Filter {
	return func(cb Handler) Handler {
		return func(ctx context.Context, args []string, chatClient chatClient) error {
			cmdCtx := UnwrapContext(ctx)

			for _, roleName := range roles {
				if hasBadge(roleName, cmdCtx.PrivMsg.User.Badges) {
					return cb(ctx, args, chatClient)
				}
			}

			return errNoPermissions
		}
	}
}

// Cooldown stops from calling a command, when not enough time passed.
func Cooldown(cooldown time.Duration) Filter {
	lastCalled := time.Time{}

	return func(cb Handler) Handler {
		return func(ctx context.Context, args []string, chatClient chatClient) error {

			if time.Now().Before(lastCalled.Add(cooldown)) {
				return errCommandOnCooldown
			}

			lastCalled = time.Now()
			return cb(ctx, args, chatClient)
		}
	}
}
