package command

import (
	"context"
	"testing"

	"github.com/gempir/go-twitch-irc/v4"
)

type chatClientMock struct{}

func (c chatClientMock) Say(channelName, message string) {

}
func (c chatClientMock) Reply(channelName, parentMessageID, message string) {

}
func (c chatClientMock) Join(channels ...string) {

}
func (c chatClientMock) Depart(channelName string) {

}

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

		ctx := setPrivateMessageToContext(context.Background(), privateMessage)
		var mockedChatClient chatClient = chatClientMock{}
		var cb Handler = func(ctx context.Context, args []string, chatClient chatClient) error {
			return nil
		}
		var expected error = errNoPermissions

		//when
		var hasRoleDecorator Handler = HasRole(decoratorRoles)(cb)
		var got error = hasRoleDecorator(ctx, args, mockedChatClient)

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
		ctx := setPrivateMessageToContext(context.Background(), privateMessage)

		var mockedChatClient chatClient = chatClientMock{}
		var cb Handler = func(ctx context.Context, args []string, chatClient chatClient) error {
			return nil
		}
		var expected error = nil

		//when
		var hasRoleDecorator Handler = HasRole(decoratorRoles)(cb)
		var got error = hasRoleDecorator(ctx, args, mockedChatClient)

		//then
		if expected != got {
			t.Errorf("Expected `%v`, got `%v` error", expected, got)
		}
	})
}
