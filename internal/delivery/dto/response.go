package dto

type ApiResponse[T any] struct {
	Message string `json:"message"`

	Data T `json:"data"`
}

type Response = ApiResponse[any]
