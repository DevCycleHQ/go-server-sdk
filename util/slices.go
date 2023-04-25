package util

import "github.com/devcyclehq/go-server-sdk/v2/api"

func ChunkSlice(slice []api.DVCEvent, chunkSize int) [][]api.DVCEvent {
	var chunks [][]api.DVCEvent
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}
