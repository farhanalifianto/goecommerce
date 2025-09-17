package model

type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Email    string `gorm:"uniqueIndex" json:"email"`
	Password string `json:"-"`
	Name     string `json:"name"`
	Role     string `json:"role"` // "user" or "admin"
}
