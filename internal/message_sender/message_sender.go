package messagesender

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ChatClient specifies a method for broadcasting messages on a channel.
type chatClient interface {
	Say(channelName, message string)
}

// MessageSender represents a struct for broadcasting messages on a channel,
// holding essential information for broadcasting.
type MessageSender struct {
	logger      *zap.Logger   // Logger used for logging.
	messages    []string      // Messages represents a list of messages, the messages are intended to be sent in the chat.
	chatClient  chatClient    // ChatClient describes a method for broadcasting messages on a channel.
	interval    time.Duration // Interval indicates how often a message should be sent.
	channelName string        // ChannelName represents a name for a channel, on where messages are sent on.
}

// New creates an instance of MessageSender for regularly sending messages in the chat.
func New(interval time.Duration, channelName string, chatMessageSender chatClient, logger *zap.Logger) *MessageSender {
	return &MessageSender{interval: interval, channelName: channelName, chatClient: chatMessageSender, logger: logger}
}

// AddMessages adds a message to the list of broadcasted messages.
func (ms *MessageSender) AddMessages(message ...string) {
	ms.messages = append(ms.messages, message...)
}

// Start runs a cron job for posting messages on the chat. This method blocks the execution of your code,
// use Goroutine with this method.
func (ms *MessageSender) Start(ctx context.Context) {
	if len(ms.messages) == 0 {
		ms.logger.Error("the list of messages in your MessageSender is empty")
		return
	}

	t := time.NewTicker(ms.interval)
	i := 0

	defer func() {
		err := ms.logger.Sync()
		if err != nil {
			fmt.Printf("sync method in logger threw an error: %v", err)
		}
	}()

	for {
		select {
		case <-t.C:
			i %= len(ms.messages)
			ms.chatClient.Say(ms.channelName, ms.messages[i])

			ms.logger.Info("send a message to the chat", zap.Int("messageIndex", i))
			i++
		case <-ctx.Done():
			ms.logger.Info("message sender ended it's job")
			return
		}
	}
}
