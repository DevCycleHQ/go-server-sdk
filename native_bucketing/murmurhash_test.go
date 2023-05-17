package native_bucketing

// test for murmurhashv3
import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMurmurhashV3(t *testing.T) {
	input := "test"
	hash := murmurhashV3(input, baseSeed)
	require.Equal(t, uint32(0x99c02ae2), hash)
}
