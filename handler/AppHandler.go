package handler

import (
	"gopkg.in/boj/redistore.v1"
	"gorm.io/gorm"
)

type appContext struct {
	Store *redistore.RediStore
	DB    *gorm.DB
}

type AppHandler struct {
	context *appContext
}
