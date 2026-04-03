package entity

// PendingRegistration — данные саморегистрации до подтверждения email (таблица pending_registrations).
type PendingRegistration struct {
	Email        string
	Login        string
	PasswordHash string
	LastName     string
	FirstName    string
	MiddleName   string
	Phone        string
	City         string
	Street       string
	House        string
	Apartment    string
	Role         string
}
