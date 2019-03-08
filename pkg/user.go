package root

type User struct {
	Id           string  `json:"id"`
	Email     string  `json:"email"`
	Password     string  `json:"password"`
}

type UserService interface {
	CreateUser(u *User) error
	GetByUsername(username string) (*User,error)
}