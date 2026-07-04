package api

import "time"

type createTaskRequest struct {
	Query string `json:"query"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}
