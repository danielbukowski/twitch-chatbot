package command

import (
	"testing"

	"github.com/gempir/go-twitch-irc/v4"
)

func TestHasRole(t *testing.T) {
	t.Run("returns ErrNoPermissions, when no roles were passed to the decorator", func(t *testing.T) {
		//given
		decoratorRoles := []string{}
		args := []string{}
		privateMessage := &twitch.PrivateMessage{
			User: twitch.User{
				Badges: map[string]int{
					"broadcaster": 1,
				},
			},
		}
		ircClient := &twitch.Client{}
		cb := func(args []string, privateMessage *twitch.PrivateMessage, ircClient *twitch.Client) error {
			return nil
		}
		var expected error = errNoPermissions

		//when
		var hasRoleDecorator CallbackSignature = HasRole(decoratorRoles, cb)
		var got error = hasRoleDecorator(args, privateMessage, ircClient)

		//then
		if expected != got {
			t.Errorf("Expected `%v`, got `%v` error", expected, got)
		}
	})

	t.Run("returns nil, when user has a role to call a command", func(t *testing.T) {
		//given
		decoratorRoles := []string{"broadcaster", "vip"}
		args := []string{}
		privateMessage := &twitch.PrivateMessage{
			User: twitch.User{
				Badges: map[string]int{
					"vip": 1,
				},
			},
		}
		ircClient := &twitch.Client{}
		cb := func(args []string, privateMessage *twitch.PrivateMessage, ircClient *twitch.Client) error {
			return nil
		}
		var expected error = nil

		//when
		var hasRoleDecorator CallbackSignature = HasRole(decoratorRoles, cb)
		var got error = hasRoleDecorator(args, privateMessage, ircClient)

		//then
		if expected != got {
			t.Errorf("Expected `%v`, got `%v` error", expected, got)
		}
	})
}
