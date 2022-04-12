package routes

import (
	controller "go-restaurant-management/controllers"
	"go-restaurant-management/middleware"

	"github.com/gin-gonic/gin"
)

func UserRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.GET("/users/:user_id", middleware.Authentication(), controller.GetUser())
	incomingRoutes.GET("/users", middleware.Authentication(), controller.GetUsers())
	incomingRoutes.POST("/users/login", controller.Login())
	incomingRoutes.POST("/users/signup", controller.Signup())
}
