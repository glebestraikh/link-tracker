package stackoverflow

type apiResponse struct {
	Items []questionItem `json:"items"`
}

type questionItem struct {
	LastActivityDate int64 `json:"last_activity_date"`
}
