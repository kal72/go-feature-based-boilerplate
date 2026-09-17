package dto

// CreateUserRequest carries the input data for creating a new user.
type CreateUserRequest struct {
	Name     string `validate:"required,min=2,max=100"`
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=8"`
}

// UpdateUserRequest carries the input data for updating an existing user.
type UpdateUserRequest struct {
	ID       uint
	CallerID uint
	Name     string `validate:"omitempty,min=2,max=100"`
}

// FindUserRequest carries the input for a single-user lookup.
type FindUserRequest struct {
	ID uint `validate:"required"`
}

// ListUsersRequest carries pagination input for listing users.
type ListUsersRequest struct {
	Page     int `validate:"min=1"`
	PageSize int `validate:"min=1,max=100"`
}
