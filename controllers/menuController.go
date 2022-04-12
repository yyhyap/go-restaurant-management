package controllers

import (
	"context"
	"go-restaurant-management/database"
	"go-restaurant-management/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var menuCollection *mongo.Collection = database.OpenCollection(database.Client, "menu")

func GetMenus() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, err := menuCollection.Find(ctx, bson.M{})
		defer cancel()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while retrieving menus from database"})
			return
		}

		var allMenus []bson.M

		if err = result.All(ctx, &allMenus); err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, allMenus)
	}
}

func GetMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		menuId := c.Param("menu_id")
		var menu models.Menu

		err := menuCollection.FindOne(ctx, bson.M{"menu_id": menuId}).Decode(&menu)
		defer cancel()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "menu not found"})
			return
		}

		c.JSON(http.StatusOK, menu)
	}
}

func CreateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {

		var menu models.Menu

		if err := c.BindJSON(&menu); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validationError := validate.Struct(menu)

		if validationError != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationError.Error()})
			return
		}

		var err error

		menu.Created_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing created_at"})
			return
		}

		menu.Updated_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing updated_at"})
			return
		}

		menu.ID = primitive.NewObjectID()
		menu.Menu_id = menu.ID.Hex()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, insertErr := menuCollection.InsertOne(ctx, menu)
		defer cancel()

		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "menu item is not created due to some errors"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func UpdateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		var menu models.Menu

		if err := c.BindJSON(&menu); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		menuId := c.Param("menu_id")

		var updateObj primitive.D

		// start_date and end_date can use nil because using pointer in struct
		if menu.Start_date != nil && menu.End_date != nil {
			if !inTimeSpan(*menu.Start_date, *menu.End_date, time.Now()) {
				msg := "kindly retype the time"
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
				return
			}
		}

		updateObj = append(updateObj, bson.E{Key: "start_date", Value: menu.Start_date})
		updateObj = append(updateObj, bson.E{Key: "end_date", Value: menu.End_date})

		if menu.Name != "" {
			updateObj = append(updateObj, bson.E{Key: "name", Value: menu.Name})
		}

		if menu.Category != "" {
			updateObj = append(updateObj, bson.E{Key: "category", Value: menu.Category})
		}

		var err error
		menu.Updated_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing updated_at"})
			return
		}

		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: menu.Updated_at})

		upsert := true
		filter := bson.M{"menu_id": menuId}
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, err := menuCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{Key: "&set", Value: updateObj},
			},
			&opt,
		)
		defer cancel()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Menu updated failed"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func inTimeSpan(start, end, now time.Time) bool {
	return (start.After(time.Now()) && end.After(start))
}
