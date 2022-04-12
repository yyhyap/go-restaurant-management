package routes

import (
	controller "go-restaurant-management/controllers"

	"github.com/gin-gonic/gin"
)

func TableRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.GET("/tables/:table_id", controller.GetTable())
	incomingRoutes.GET("/tables", controller.GetTables())
	incomingRoutes.POST("/tables", controller.CreateTable())
	incomingRoutes.PATCH("/tables/:table_id", controller.UpdateTable())
}
