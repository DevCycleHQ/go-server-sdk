package api

func ChunkSlice(slice []DVCEvent, chunkSize int) [][]DVCEvent {
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
