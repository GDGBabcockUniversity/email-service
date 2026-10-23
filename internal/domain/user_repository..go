package domain

type UserRepository interface {
	Save(user User) error

	FindById(id string) (User, error)

	DeleteById(id string) error

	FindByEmail(email string) (User, error)
}
