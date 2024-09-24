package command

import (
	"context"
	"errors"
)

var errNoPermissions = errors.New("called a command without a needed role")

// Available twitch badges:
// ["broadcaster", "moderator", "subscriber", "artist-badge", "founder", "vip", "sub-gifter", "bits", "partner", "staff"].
func hasBadge(badgeName string, badges map[string]int) bool {
	return badges[badgeName] != 0
}

func HasRole(roles []string) Filter {
	return func(cb Handler) Handler {
		return func(ctx context.Context, args []string, chatClient chatClient) error {

			privMsg := GetPrivateMessageFromContext(ctx)

			for _, roleName := range roles {
				if hasBadge(roleName, privMsg.User.Badges) {
					return cb(ctx, args, chatClient)
				}
			}

			return errNoPermissions
		}
	}
}
