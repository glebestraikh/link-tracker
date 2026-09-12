package inmemory

import (
	"context"
	"sync"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
)

type LinkRepository struct {
	mu            sync.RWMutex
	links         map[LinkID]*model.Link
	urlToID       map[URL]LinkID
	chatToLinkIDs map[ChatID]map[LinkID]struct{}
	nextID        int64
}

func NewLinkRepository() *LinkRepository {
	return &LinkRepository{
		links:         make(map[LinkID]*model.Link),
		urlToID:       make(map[URL]LinkID),
		chatToLinkIDs: make(map[ChatID]map[LinkID]struct{}),
	}
}

func (r *LinkRepository) Save(_ context.Context, link model.Link) (*model.Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(link.ChatIDs) == 0 {
		return nil, model.ErrLinkNotFound
	}

	chatID := link.ChatIDs[0]

	if existingID, ok := r.urlToID[link.URL]; ok {
		if r.isSubscribed(chatID, existingID) {
			return nil, model.ErrLinkAlreadyTracked
		}

		existing := r.links[existingID]
		existing.ChatIDs = append(existing.ChatIDs, chatID)
		r.addSubscription(chatID, existingID)

		return existing, nil
	}

	r.nextID++
	id := r.nextID
	link.ID = id
	r.links[id] = &link
	r.urlToID[link.URL] = id
	r.addSubscription(chatID, id)

	return r.links[id], nil
}

func (r *LinkRepository) Delete(_ context.Context, chatID int64, url string) (*model.Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	linkID, ok := r.urlToID[url]
	if !ok {
		return nil, model.ErrLinkNotFound
	}

	chatKey := chatID
	if !r.isSubscribed(chatKey, linkID) {
		return nil, model.ErrLinkNotFound
	}

	link := r.links[linkID]
	originalChatIDs := make([]int64, len(link.ChatIDs))
	copy(originalChatIDs, link.ChatIDs)
	original := *link
	original.ChatIDs = originalChatIDs

	link.ChatIDs = removeChatID(link.ChatIDs, chatID)
	r.removeSubscription(chatKey, linkID)

	if len(link.ChatIDs) == 0 {
		delete(r.links, linkID)
		delete(r.urlToID, url)
	}

	return &original, nil
}

func (r *LinkRepository) FindByChatID(_ context.Context, chatID int64, offset, limit uint64) ([]*model.Link, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	linkIDs := r.chatToLinkIDs[chatID]
	links := make([]*model.Link, 0, len(linkIDs))
	for linkID := range linkIDs {
		if link, ok := r.links[linkID]; ok {
			links = append(links, link)
		}
	}

	return paginate(links, offset, limit), nil
}

func (r *LinkRepository) FindAllPaged(_ context.Context, offset, limit uint64) ([]*model.Link, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	links := make([]*model.Link, 0, len(r.links))
	for _, link := range r.links {
		links = append(links, link)
	}

	return paginate(links, offset, limit), nil
}

func (r *LinkRepository) UpdateLastUpdated(_ context.Context, id int64, t time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	link, ok := r.links[id]
	if !ok {
		return model.ErrLinkNotFound
	}

	link.LastUpdated = t

	return nil
}

func (r *LinkRepository) addSubscription(chatID ChatID, linkID LinkID) {
	if _, ok := r.chatToLinkIDs[chatID]; !ok {
		r.chatToLinkIDs[chatID] = make(map[LinkID]struct{})
	}

	r.chatToLinkIDs[chatID][linkID] = struct{}{}
}

func (r *LinkRepository) removeSubscription(chatID ChatID, linkID LinkID) {
	chatLinks, ok := r.chatToLinkIDs[chatID]
	if !ok {
		return
	}

	delete(chatLinks, linkID)

	if len(chatLinks) == 0 {
		delete(r.chatToLinkIDs, chatID)
	}
}

func (r *LinkRepository) isSubscribed(chatID ChatID, linkID LinkID) bool {
	chatLinks, ok := r.chatToLinkIDs[chatID]
	if !ok {
		return false
	}
	_, ok = chatLinks[linkID]
	return ok
}

func removeChatID(chatIDs []int64, target int64) []int64 {
	for i, id := range chatIDs {
		if id == target {
			return append(chatIDs[:i], chatIDs[i+1:]...)
		}
	}

	return chatIDs
}

func paginate(links []*model.Link, offset, limit uint64) []*model.Link {
	if offset >= uint64(len(links)) {
		return []*model.Link{}
	}
	end := offset + limit
	if limit == 0 || end > uint64(len(links)) {
		end = uint64(len(links))
	}
	return links[offset:end]
}
