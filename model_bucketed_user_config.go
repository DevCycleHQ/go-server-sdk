package devcycle

type BucketedUserConfig struct {
	Project     string `json:"project"`
	Environment string `json:"environment"`
	Features    string `json:"features"`
}
