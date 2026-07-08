package request

type CreateUserRequest struct {
	FirstName string   `json:"first_name" validate:"required,min=3,max=20"`
	LastName  string   `json:"last_name" validate:"required,min=3,max=20"`
	Email     string   `json:"email" validate:"required,email"`
	ImageURL  string   `json:"image_url" validate:"optional_url"`
	Password  string   `json:"password" validate:"required,min=6"`
	Roles     []string `json:"roles" validate:"required,roles"`
}

type UpdateUserRequest struct {
	Username  string   `json:"username" validate:"required,max=60"`
	FirstName string   `json:"first_name" validate:"required,min=3,max=20"`
	LastName  string   `json:"last_name" validate:"required,min=3,max=20"`
	ImageURL  string   `json:"image_url" validate:"optional_url"`
	Roles     []string `json:"roles" validate:"required,roles"`
}
