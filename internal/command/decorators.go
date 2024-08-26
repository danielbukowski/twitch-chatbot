package command

import (
	"errors"

	"github.com/gempir/go-twitch-irc/v4"
)

var errNoPermissions = errors.New("called a command without a needed role")

// Available twitch badges:
// ["broadcaster", "moderator", "subscriber", "artist-badge", "founder", "vip", "sub-gifter", "bits", "partner", "staff"].
func hasBadge(badgeName string, badges map[string]int) bool {
	return badges[badgeName] != 0
}

func HasRole(roles []string, cb CallbackSignature) CallbackSignature {
	return func(args []string, privateMessage *twitch.PrivateMessage, ircClient *twitch.Client) error {
		for _, roleName := range roles {
			if hasBadge(roleName, privateMessage.User.Badges) {
				return cb(args, privateMessage, ircClient)
			}
		}

		return errNoPermissions
	}
}
