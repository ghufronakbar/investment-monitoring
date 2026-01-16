// internal/module/iam/profile/service/params.go
package profile_service

// UpdateProfileParams defines the request body for updating profile
type UpdateProfileParams struct {
	Name     string  `json:"name" validate:"required,min=2,max=100"`
	ImageUrl *string `json:"imageUrl" validate:"omitempty,url"`
}

// ChangePasswordParams defines the request body for changing password
type ChangePasswordParams struct {
	CurrentPassword string `json:"currentPassword" validate:"required,min=6"`
	NewPassword     string `json:"newPassword" validate:"required,min=6"`
}
