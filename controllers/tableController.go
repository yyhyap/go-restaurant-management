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

var tableCollection *mongo.Collection = database.OpenCollection(database.Client, "table")

func GetTables() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, err := tableCollection.Find(ctx, bson.M{})
		defer cancel()

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while retrieving tables from database"})
			return
		}

		var allTables []bson.M

		if err := result.All(ctx, &allTables); err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, allTables)
	}
}

func GetTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		tableId := c.Param("table_id")
		var table models.Table

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		err := tableCollection.FindOne(ctx, bson.M{"table_id": tableId}).Decode(&table)
		defer cancel()

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "table not found"})
			return
		}

		c.JSON(http.StatusOK, table)
	}
}

func CreateTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		var table models.Table

		if err := c.BindJSON(&table); err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validationError := validate.Struct(table)

		if validationError != nil {
			log.Println(validationError)
			c.JSON(http.StatusBadRequest, gin.H{"error": validationError.Error()})
			return
		}

		var err error

		table.Created_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing created_at"})
			return
		}

		table.Updated_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing updated_at"})
			return
		}

		table.ID = primitive.NewObjectID()
		table.Table_id = table.ID.Hex()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, insertErr := tableCollection.InsertOne(ctx, table)
		defer cancel()

		if insertErr != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "table is not created"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func UpdateTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		tableId := c.Param("table_id")
		var table models.Table

		if err := c.BindJSON(&table); err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var updateObj primitive.D

		if table.Number_of_guest != nil {
			updateObj = append(updateObj, bson.E{Key: "number_of_guest", Value: table.Number_of_guest})
		}

		if table.Table_number != nil {
			updateObj = append(updateObj, bson.E{Key: "table_number", Value: table.Table_number})
		}

		var err error

		table.Updated_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing updated_at"})
			return
		}

		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: table.Updated_at})

		upsert := true
		filter := bson.M{"table_id": tableId}
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, err := tableCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{Key: "$set", Value: updateObj},
			},
			&opt,
		)
		defer cancel()

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "table updated failed"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
