package user

// userResponse is the public user representation shared across the feature's endpoints.
type userResponse struct {
	ID       string `json:"id"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
}

// sessionResponse is the response of endpoints that create a session (register, login).
type sessionResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}
