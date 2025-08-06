package dto

type LoginRequest struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
}
