package api

type Project struct {
	Id               string          `json:"_id" validate:"required"`
	Key              string          `json:"key" validate:"required"`
	A0OrganizationId string          `json:"a0_organization" validate:"required"`
	Settings         ProjectSettings `json:"settings" validate:"required"`
}

type ProjectSettings struct {
	EdgeDB                     EdgeDBSettings `json:"edgeDB"`
	OptIn                      OptInSettings  `json:"optIn"`
	DisablePassthroughRollouts bool           `json:"disablePassthroughRollouts"`
}

type EdgeDBSettings struct {
	Enabled bool `json:"enabled"`
}

type OptInSettings struct {
	Enabled     bool        `json:"enabled"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	ImageURL    string      `json:"imageURL"`
	Colors      OptInColors `json:"colors"`
}

type OptInColors struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
}
