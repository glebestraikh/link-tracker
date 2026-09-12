package httpadapter

type addLinkRequest struct {
	Link string   `json:"link"`
	Tags []string `json:"tags"`
}

type removeLinkRequest struct {
	Link string `json:"link"`
}

type linkResponse struct {
	ID   int64    `json:"id"`
	URL  string   `json:"url"`
	Tags []string `json:"tags"`
}

type listLinksResponse struct {
	Links []linkResponse `json:"links"`
	Size  int32          `json:"size"`
}

type apiErrorResponse struct {
	Description string `json:"description"`
}
