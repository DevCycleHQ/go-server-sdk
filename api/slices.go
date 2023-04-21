package api

func ChunkSlice(slice []Event, chunkSize int) [][]Event {
	if chunkSize <= 0 {
		chunkSize = 1
	}
	var chunks [][]Event
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}
