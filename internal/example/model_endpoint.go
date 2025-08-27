package example

// apiCreateUserRequest represents the request body structure for creating
// a new user through the API. This structure defines the expected JSON
// format that the API endpoint requires for user creation operations.
type apiCreateUserRequest struct {
	UserName string `json:"user_name"`
}

// apiCreateUserResponse represents the response body structure returned
// by the API when a new user is successfully created. This structure
// defines the JSON format that the API returns, typically containing
// the newly assigned user ID and other relevant creation metadata.
type apiCreateUserResponse struct {
	UserID uint64 `json:"user_id"`
}

// apiGetUserByIDResponse represents the response body structure returned
// by the API when retrieving a user by their ID. This structure defines
// the complete user data format as provided by the API endpoint.
type apiGetUserByIDResponse struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

// ToModel converts the API response structure to a domain model object,
// providing type-safe transformation between the external API format
// and internal application representations.
func (resp apiGetUserByIDResponse) ToModel() *User {
	return &User{
		ID:   UserID(resp.ID),
		Name: resp.Name,
	}
}
