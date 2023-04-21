package api

type Environment struct {
	Id  string `json:"_id" validate:"required"`
	Key string `json:"key" validate:"required"`
}
