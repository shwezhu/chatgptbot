package model

// User Declaring Models: https://gorm.io/docs/models.html
type User struct {
	ID       uint   `gorm:"column:user_id;autoIncrement;primaryKey;"`
	Username string `gorm:"type:varchar(16);column:username;not null;"`
	Password string `gorm:"type:varchar(20);column:password;not null;"`
}

// TableName Change the default table name by implementing the Tabler interface
// https://gorm.io/docs/conventions.html#Temporarily-specify-a-name
func (User) TableName() string {
	return "user"
}
