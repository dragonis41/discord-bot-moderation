package utils

// SplitMessage splits a long message into multiple parts based on the limit
func SplitMessage(message string, limit int) []string {
	if len(message) <= limit {
		return []string{message}
	}

	var messages []string
	runes := []rune(message)

	for len(runes) > 0 {
		if len(runes) <= limit {
			messages = append(messages, string(runes))
			break
		}

		splitIndex := limit

		// Try to find a good split point (newline, space, or punctuation)
		for i := limit - 1; i > limit-200 && i > 0; i-- {
			if runes[i] == '\n' || runes[i] == ' ' || runes[i] == '.' || runes[i] == ',' {
				splitIndex = i + 1
				break
			}
		}

		messages = append(messages, string(runes[:splitIndex]))
		runes = runes[splitIndex:]
	}

	return messages
}
