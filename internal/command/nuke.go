package command

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	lg "github.com/danielbukowski/twitch-chatbot/internal/logger"
	messagestorage "github.com/danielbukowski/twitch-chatbot/internal/message/storage"
	"github.com/nicklaw5/helix/v2"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type NukeCommand struct {
	messageStorage *messagestorage.MessageStorage
	helixClient    *helix.Client
}

func NewNuke(messageStorage *messagestorage.MessageStorage, helixClient *helix.Client) *NukeCommand {
	return &NukeCommand{
		messageStorage: messageStorage,
		helixClient:    helixClient,
	}
}

// Nuke timeouts users if one of theirs messages are the keyword, that you give to the command.
// Command signature: <prefix>nuke <keyword> <duration> <unit>
// Available units: s, m, h, and d.
// The pattern matching to the keyword is case-insensitive.
// Example: '!nuke buh 5 m' - timeouts users for 5 minutes, if they wrote ' buh ' in theirs messages.
func (np *NukeCommand) Nuke(ctx context.Context, args []string, _ chatClient) error {
	keyword := args[0]
	duration, err := strconv.Atoi(args[1])
	if err != nil {
		return errors.Join(errors.New("failed to convert the duration"), err)
	}
	unit := args[2]
	timeoutInSeconds := 0

	commandCtx := UnwrapContext(ctx)

	// current := time.Now()

	switch unit {
	case "s":
		timeoutInSeconds = duration
	case "m":
		timeoutInSeconds = 60 * duration
	case "h":
		timeoutInSeconds = 3_600 * duration
	case "d":
		timeoutInSeconds = 86_400 * duration
	default:
		return fmt.Errorf("got an unknown time unit")
	}

	rgxp, err := regexp.Compile(fmt.Sprintf(`(?i)\b%s\b`, keyword))
	if err != nil {
		return errors.Join(errors.New("failed to compile the regexp"), err)
	}

	messages := np.messageStorage.Messages()
	timeouts := make(map[string]bool, len(messages))

	defer func() {
		lg.Flush(commandCtx.Logger)
	}()

	// TODO: filter out vips, moderators and the broadcaster from the message
	for i, message := range messages {

		separatorIndex := strings.Index(message, ":")
		if separatorIndex == -1 {
			commandCtx.Logger.Debug("got an empty message in the messages, breaking the loop", zap.Int("messageIndex", i))
			break
		}

		username := message[:separatorIndex]
		messageContent := message[separatorIndex:]

		_, ok := timeouts[username]
		if ok {
			commandCtx.Logger.Debug("skipping a message, user already added to the timeouts", zap.String("username", username))
			continue
		}

		isMatch := rgxp.MatchString(messageContent)

		if isMatch {
			commandCtx.Logger.Debug("user's message contains the keyword, adding a user to the timeouts",
				zap.String("username", username),
				zap.String("message", messageContent))

			timeouts[username] = true
		}
	}

	// np.logger.Debug("calculated how many people are getting a timeout",
	// 	zap.Int("peopleToTimeout", len(timeouts)),
	// 	zap.Int("messages", len(messages)))

	// users := make([]string, 0, len(timeouts))

	// lets re-user the messages to avoid allocation. Why? ...
	usersToTimeout := messages[:0]
	for userName := range timeouts {
		usersToTimeout = append(usersToTimeout, userName)
	}

	// You can get info about max 100 users at 1 time
	res, err := np.helixClient.GetUsers(&helix.UsersParams{
		Logins: usersToTimeout,
	})
	if err != nil {
		return errors.Join(errors.New("failed to get information about twitch users"), err)
	}

	if res.ErrorStatus != 200 {
		return errors.New("failed to get information about twitch users")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	g, _ := errgroup.WithContext(ctx)

	maxConcurrentRequests := 20
	g.SetLimit(maxConcurrentRequests)

	//TODO: get somehow ids of my channel and chatbot
	for _, u := range res.Data.Users {
		g.Go(func() error {
			// TODO: get twitch token with scope: moderator:manage:banned_users
			_, err := np.helixClient.BanUser(&helix.BanUserParams{
				BroadcasterID: "",
				ModeratorId:   "",
				Body: helix.BanUserRequestBody{
					UserId:   u.ID,
					Reason:   "u got nuked!",
					Duration: timeoutInSeconds,
				},
			})

			return err
		})
	}

	err = g.Wait()
	if err != nil {
		return errors.Join(errors.New("failed to timeout twitch users"), err)
	}

	// fmt.Printf("It took %v", time.Since(current))
	// chatClient.Say(privMsg.Channel, "Done the job!")
	return nil
}

// TODO: reword this

// ValidateNukeArguments processes the passed arguments to the nuke command.
func ValidateNukeArguments() Filter {
	return func(cb Handler) Handler {
		return func(ctx context.Context, args []string, chatClient chatClient) error {
			if len(args) != 3 {
				return fmt.Errorf("the number of arguments should be equal to 3, got %d)", len(args))
			}

			duration := args[1]
			unit := args[2]

			switch unit {
			case "s", "m", "h", "d":
				break
			default:
				return fmt.Errorf("time unit should be equal to: s, m, h or d")
			}

			d, err := strconv.Atoi(duration)
			if err != nil {
				return fmt.Errorf("duration should be a number")
			}

			if d <= 0 {
				return fmt.Errorf("duration should be positive")
			}

			return cb(ctx, args, chatClient)
		}
	}
}
