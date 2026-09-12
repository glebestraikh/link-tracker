package github

import "time"

type repoResponse struct {
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	PushedAt  *time.Time `json:"pushed_at,omitempty"`
}
