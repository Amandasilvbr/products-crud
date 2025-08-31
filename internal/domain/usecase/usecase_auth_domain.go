package usecase

// AuthUsecaseInterface defines the interface for authentication-related use cases
type AuthUsecaseInterface interface {
	Login(email, password string) (string, error)
	CreateUser(name, email, password string) error
}
