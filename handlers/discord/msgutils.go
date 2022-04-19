package discord

import (
	"fmt"
	"strings"
)

// DiscordMaxDescriptionLength is maximum length of description field in 1 message.
// It can be used as maxLength parameter for SplitMessageParts function.
const DiscordMaxDescriptionLength = 4096

// SplitMessageParts splits message parts by max length allowed
// by discord into list of messages to be sent.
func SplitMessageParts(slice []string, maxLength int) []string {
	var messages []string

	var (
		len int
		buf strings.Builder
	)
	for _, part := range slice {
		partWithNewline := fmt.Sprintf("%s\n", part)
		partCharacterCount := strings.Count(partWithNewline, "")
		// If current length of string + this new part is over the maximum
		// append buf to the messages out, reset buf and len, and start
		// with this part as the 1st item in the reset buf and len.
		if len+partCharacterCount > maxLength {
			messages = append(messages, buf.String())
			buf.Reset()
			buf.WriteString(partWithNewline)
			len = partCharacterCount
		} else {
			buf.WriteString(partWithNewline)
			len += partCharacterCount
		}
	}
	// Make sure to append the buf to the messages in the end.
	messages = append(messages, buf.String())
	return messages
}
