package utils

// SplitText splits a long string into chunks of approximately 'chunkSize' characters.
// It includes an 'overlap' to preserve context at boundaries.
// This is a simple character-based splitter. Ideally, use a tokenizer-aware splitter.
func SplitText(text string, chunkSize int, overlap int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	runes := []rune(text)
	totalLen := len(runes)

	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize // fallback if overlap >= chunkSize
	}

	for i := 0; i < totalLen; i += step {
		end := i + chunkSize
		if end > totalLen {
			end = totalLen
		}

		// Optimization: Try to break at a newline or space if close to 'end'
		// (This is a naive improvement to avoid cutting words in half)
		// For now, strict character slicing is safer than losing data.

		chunks = append(chunks, string(runes[i:end]))

		if end == totalLen {
			break
		}
	}

	return chunks
}
