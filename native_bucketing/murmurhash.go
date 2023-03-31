package native_bucketing

import "github.com/spaolacci/murmur3"

func Murmurhashv3(data []byte, seed uint32) uint32 {
	return murmur3.Sum32WithSeed(data, seed)
}
