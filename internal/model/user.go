// Package model provides database models.
package model

// UserStatus represents the status of a user.
const (
	// UserStatusNormal represents a normal/active user.
	UserStatusNormal int = 1
	// UserStatusCancelled represents a cancelled user.
	UserStatusCancelled int = 2
	// UserStatusLocked represents a locked user.
	UserStatusLocked int = 3
)

// UserGender represents the gender of a user.
const (
	// UserGenderUnknown represents unknown gender.
	UserGenderUnknown int = 0
	// UserGenderMale represents male.
	UserGenderMale int = 1
	// UserGenderFemale represents female.
	UserGenderFemale int = 2
)

// User represents a user in the system.
type User struct {
	BaseModel
	Name     string  `gorm:"size:100;not null" json:"name"`
	Email    *string `gorm:"size:255;uniqueIndex" json:"email,omitempty"`
	Phone    *string `gorm:"size:20;uniqueIndex" json:"phone,omitempty"`
	Password string  `gorm:"size:255;not null" json:"-"` // 密码不序列化到 JSON
	Gender   int     `gorm:"not null;default:0" json:"gender"`
	Avatar   *string `gorm:"size:500" json:"avatar,omitempty"`
	Status   int     `gorm:"not null;default:1" json:"status"`
}

// TableName specifies the table name for User model.
func (User) TableName() string {
	return "users"
}

// IsNormal returns true if the user status is normal.
func (u *User) IsNormal() bool {
	return u.Status == UserStatusNormal
}

// IsCancelled returns true if the user status is cancelled.
func (u *User) IsCancelled() bool {
	return u.Status == UserStatusCancelled
}

// IsLocked returns true if the user status is locked.
func (u *User) IsLocked() bool {
	return u.Status == UserStatusLocked
}

// HasEmailOrPhone returns true if the user has either email or phone set.
// This is useful for validation since users should be able to login with either email or phone.
func (u *User) HasEmailOrPhone() bool {
	return (u.Email != nil && *u.Email != "") || (u.Phone != nil && *u.Phone != "")
}

// GetEmail returns the email value or empty string if nil.
func (u *User) GetEmail() string {
	if u.Email == nil {
		return ""
	}
	return *u.Email
}

// GetPhone returns the phone value or empty string if nil.
func (u *User) GetPhone() string {
	if u.Phone == nil {
		return ""
	}
	return *u.Phone
}

// IsMale returns true if the user gender is male.
func (u *User) IsMale() bool {
	return u.Gender == UserGenderMale
}

// IsFemale returns true if the user gender is female.
func (u *User) IsFemale() bool {
	return u.Gender == UserGenderFemale
}

// GetAvatar returns the avatar URL or empty string if nil.
func (u *User) GetAvatar() string {
	if u.Avatar == nil {
		return ""
	}
	return *u.Avatar
}
