package main

import (
	helper "go-restaurant-management/helpers"
	"go-restaurant-management/middleware"
	"go-restaurant-management/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	port := helper.GetEnvVariable("PORT")

	if port == "" {
		port = "8000"
	}

	router := gin.New()
	router.Use(gin.Logger())

	routes.UserRoutes(router)
	router.Use(middleware.Authentication())

	routes.FoodRoutes(router)
	routes.MenuRoutes(router)
	routes.TableRoutes(router)
	routes.OrderRoutes(router)
	routes.OrderItemRoutes(router)
	routes.InvoiceRoutes(router)

	router.Run(":" + port)
}
