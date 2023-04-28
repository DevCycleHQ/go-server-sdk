package api

func ChunkSlice(slice []DVCEvent, chunkSize int) [][]DVCEvent {
	if chunkSize <= 0 {
		chunkSize = 1
	}
	var chunks [][]DVCEvent
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}
