package domain

import "github.com/google/uuid"

type Tenant struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ConcurrencyConfig struct {
	Workers int `json:"workers"`
}
