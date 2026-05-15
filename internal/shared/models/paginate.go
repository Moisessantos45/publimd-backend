package models


type Pagination struct {
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
}
