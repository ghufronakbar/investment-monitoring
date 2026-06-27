// internal/module/iam/auth/service/params.go
package auth_service

type CheckEmailParams struct {
	Email string `json:"email" validate:"required,email"`
}

type LoginParams struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type SetPasswordParams struct {
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirmPassword" validate:"required,eqfield=Password"`
}

type RefreshTokenParams struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type LogoutParams struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type RevokeAllParams struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}
