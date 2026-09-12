package httpadapter

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"

type apiErrorResponse struct {
	Description string `json:"description"`
}

type updateRequest struct {
	ID          int64   `json:"id"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
	TgChatIDs   []int64 `json:"tgChatIds"`
}

func (r updateRequest) toModel() model.LinkUpdate {
	return model.LinkUpdate{
		ID:          r.ID,
		URL:         r.URL,
		Description: r.Description,
		TgChatIDs:   r.TgChatIDs,
	}
}
