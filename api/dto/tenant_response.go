package dto

type CreateTenantResponse struct {
	ID   string `json:"id" example:"a1b2c3d4-e5f6-7890-1234-567890abcdef"`
	Name string `json:"name" example:"My Awesome Tenant"`
}

type MessageResponse struct {
	Message string `json:"message" example:"operation successful"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"resource not found"`
}
