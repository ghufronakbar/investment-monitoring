package pagination

type PaginationParams struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

type Pagination struct {
	Total       int  `json:"total"`
	Page        int  `json:"page"`
	Limit       int  `json:"limit"`
	TotalPages  int  `json:"totalPages"`
	HasNextPage bool `json:"hasNextPage"`
	HasPrevPage bool `json:"hasPrevPage"`
}

func NewPagination(params *PaginationParams) Pagination {
	total := params.Total
	page := params.Page
	limit := params.Limit

	if page < 1 {
		page = 1
	}

	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit // ceil(total/limit)
	}

	return Pagination{
		Total:       total,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
		HasNextPage: page < totalPages,
		HasPrevPage: page > 1,
	}
}
