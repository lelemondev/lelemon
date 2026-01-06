package entity

// Page represents a paginated result
type Page[T any] struct {
	Data   []T `json:"Data"`
	Total  int `json:"Total"`
	Limit  int `json:"Limit"`
	Offset int `json:"Offset"`
}
