package token

import "time"

type Payload struct {
	TokenID   string    `json:"token_id"`
	Username  string    `json:"username"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}
