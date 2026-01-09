package entity

import "time"

type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash *string // nil for OAuth-only users
	GoogleID     *string // OAuth provider ID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserUpdate struct {
	Name         *string
	PasswordHash *string
	GoogleID     *string
}

type NewUser struct {
	Email        string
	Name         string
	PasswordHash *string
	GoogleID     *string
}

// GetID returns the user ID (implements common user interface)
func (u *User) GetID() string {
	return u.ID
}

// GetEmail returns the user email (implements common user interface)
func (u *User) GetEmail() string {
	return u.Email
}
