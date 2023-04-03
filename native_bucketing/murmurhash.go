package native_bucketing

import "github.com/spaolacci/murmur3"

func murmurhashV3(data []byte, seed uint32) uint32 {
	mh := murmur3.Sum32WithSeed(data, seed)
	return mh
}
