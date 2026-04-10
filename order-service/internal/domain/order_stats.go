package domain

type OrderStats struct {
	Total     int64 `json:"total"`
	Pending   int64 `json:"pending"`
	Paid      int64 `json:"paid"`
	Failed    int64 `json:"failed"`
	Cancelled int64 `json:"cancelled"`
}
