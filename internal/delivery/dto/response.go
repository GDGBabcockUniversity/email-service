package dto

type ApiResponse[T any] struct {
	Message string `json:"message"`

	Data T `json:"data"`
}

type Response = ApiResponse[any]

// EmailStats is the delivery breakdown shown on the admin dashboard.
type EmailStats struct {
	Total      int64 `json:"total"`
	Pending    int64 `json:"pending"`
	Processing int64 `json:"processing"`
	Sent       int64 `json:"sent"`
	Failed     int64 `json:"failed"`
}
