package model

import "time"

type DeviceCode struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	UserID    string    `json:"user_id"`
	Claimed   bool      `json:"claimed"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
