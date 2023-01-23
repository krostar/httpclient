package example

type apiCreateUserRequest struct {
	UserName string `json:"user_name"`
}

type apiCreateUserResponse struct {
	UserID uint64 `json:"user_id"`
}

type apiGetUserByIDResponse struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

func (resp apiGetUserByIDResponse) ToModel() *User {
	return &User{
		ID:   UserID(resp.ID),
		Name: resp.Name,
	}
}
