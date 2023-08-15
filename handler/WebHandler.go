package handler

import (
	"fmt"
	"gorm.io/gorm"
	. "gptbot/model"
	"net/http"
	"time"
)

// IndexHandler Function name starts with an uppercase letter: Public function
// And name starts with a lowercase letter: Private function
func IndexHandler(db *gorm.DB) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

	})
}

func LoginHandler(db *gorm.DB) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

	})
}

func RegisterHandler(db *gorm.DB) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user := User{Username: "david", Password: "778899d"}

		if !db.Migrator().HasTable(&User{}) {
			// Migrate the schema - create table
			err := db.AutoMigrate(&User{})
			if err != nil {
				panic("failed to migrate the user schema")
			}
		}

		// Don't need to switch table before insert, gorm will switch table according to the value(object) of struct
		// How gorm find table by the struct: https://gorm.io/docs/conventions.html#Pluralized-Table-Name
		// if you really want, you can write like this: webHandler.DB.Table("users").Create()
		// https://gorm.io/docs/conventions.html#Temporarily-specify-a-name
		err := db.Create(&user).Error
		if err != nil {
			panic(fmt.Errorf("failed to insert user: %v at %v", user.Username, time.Now()))
		}
	})
}
