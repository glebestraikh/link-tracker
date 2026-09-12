package model

type StateType int

const (
	StateWaitingURL StateType = iota + 1
	StateWaitingTags
	StateWaitingUntrackURL
)

type UserState struct {
	Type StateType
	URL  string
}
