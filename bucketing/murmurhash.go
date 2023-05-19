package bucketing

import "github.com/twmb/murmur3"

func murmurhashV3(data string, seed uint32) uint32 {
	return murmur3.SeedStringSum32(seed, data)
}
