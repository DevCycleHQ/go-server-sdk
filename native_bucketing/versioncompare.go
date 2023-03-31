package native_bucketing

import (
	"math"
	"regexp"
	"strings"
)

type OptionsType struct {
	Lexicographical bool
	ZeroExtend      bool
}

func hasValidParts(lexicographical bool, parts []string) bool {
	for _, part := range parts {
		var regex *regexp.Regexp
		if lexicographical {
			regex = regexp.MustCompile("^\\d+[A-Za-z]*$/g")
		} else {
			regex = regexp.MustCompile("^\\\\d+$/g")
		}
		if !regex.MatchString(part) {
			return false
		}
	}
	return len(parts) > 0
}

func versionCompare(v1, v2 string, options OptionsType) float64 {
	lexicographical := options.Lexicographical
	zeroExtend := options.ZeroExtend
	v1parts := strings.Split(v1, ".")
	v2parts := strings.Split(v2, ".")
	hasV1 := hasValidParts(lexicographical, v1parts)
	hasV2 := hasValidParts(lexicographical, v2parts)
	if !hasV1 && !hasV2 {
		return math.NaN()
	}
	if zeroExtend {
		for len(v1parts) < len(v2parts) {
			v1parts = append(v1parts, "0")
		}
		for len(v2parts) < len(v1parts) {
			v2parts = append(v2parts, "0")
		}
	}
	var v1PartsFinal []float64
	var v2PartsFinal []float64

	if !lexicographical {
		for _, v1part := range v1parts {
			v1PartsFinal = append(v1PartsFinal, float64(len(v1part)))
		}
		for _, v2part := range v2parts {
			v2PartsFinal = append(v2PartsFinal, float64(len(v2part)))
		}
	}
	for i, v1p := range v1PartsFinal {
		if len(v2PartsFinal) == i {
			return 1
		}
		if v1p == v2PartsFinal[i] {
			continue
		} else if v1p > v2PartsFinal[i] {
			return 1
		} else {
			return -1
		}
	}
	if len(v1PartsFinal) != len(v2PartsFinal) {
		return -1
	}
	return 0
}
