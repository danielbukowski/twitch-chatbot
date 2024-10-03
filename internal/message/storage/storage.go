package messagestorage

// MessageStorage stores in memory the recent messages from the twitch chat, up to the capacity you give.
// The messages are lost, when the chatbot is shut down.
type MessageStorage struct {
	messages  []string // Messages is a list of messages from the twitch chat.
	nextIndex int      // NextIndex represents a next available index for an incoming message.
}

// New returns a new instance of MessageStore for storing temporary messages.
func New(capacity int) *MessageStorage {
	return &MessageStorage{
		messages:  make([]string, capacity),
		nextIndex: 0,
	}
}

// AddMessage insert a message to the messages.
func (ms *MessageStorage) AddMessage(message string) {
	ms.messages[ms.nextIndex] = message
	ms.nextIndex = (ms.nextIndex + 1) % len(ms.messages)
}

// Messages returns a copy of the messages.
func (ms *MessageStorage) Messages() []string {
	buffer := make([]string, len(ms.messages))
	copy(buffer, ms.messages)

	return buffer
}
