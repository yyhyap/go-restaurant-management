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

var orderCollection *mongo.Collection = database.OpenCollection(database.Client, "order")

func GetOrders() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, err := orderCollection.Find(ctx, bson.M{})
		defer cancel()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while retrieving orders from database"})
			return
		}

		var allOrders []bson.M

		if err = result.All(ctx, &allOrders); err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, allOrders)
	}
}

func GetOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Param("order_id")
		var order models.Order

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		err := orderCollection.FindOne(ctx, bson.M{"order_id": orderId}).Decode(&order)
		defer cancel()

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "order not found"})
			return
		}

		c.JSON(http.StatusOK, order)
	}
}

func CreateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var order models.Order
		var table models.Table

		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validationError := validate.Struct(order)

		if validationError != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationError.Error})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		err := tableCollection.FindOne(ctx, bson.M{"table_id": order.Table_id}).Decode(&table)
		defer cancel()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "table not found"})
			return
		}

		order.Created_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing created_at"})
			return
		}

		order.Updated_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing updated_at"})
			return
		}

		order.ID = primitive.NewObjectID()
		order.Order_id = order.ID.Hex()

		result, insertErr := orderCollection.InsertOne(ctx, order)
		defer cancel()

		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "order is not created due to some errors"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func UpdateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var order models.Order

		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		orderId := c.Param("order_id")
		var table models.Table
		var updateObj primitive.D

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		if order.Table_id != nil {
			err := tableCollection.FindOne(ctx, bson.M{"table_id": order.Table_id}).Decode(&table)
			defer cancel()

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "table not found"})
				return
			}

			updateObj = append(updateObj, bson.E{Key: "table_id", Value: order.Table_id})
		}

		var err error
		order.Updated_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing updated_at"})
			defer cancel()
			return
		}

		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: order.Updated_at})

		upsert := true
		filter := bson.M{"order_id": orderId}
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		result, err := orderCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{Key: "$set", Value: updateObj},
			},
			&opt,
		)
		defer cancel()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "order updated failed"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func OrderItemOrderCreator(order models.Order) (string, error) {

	var err error

	order.Created_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

	if err != nil {
		log.Println(err)
		return "", err
	}

	order.Updated_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

	if err != nil {
		log.Println(err)
		return "", err
	}

	order.ID = primitive.NewObjectID()
	order.Order_id = order.ID.Hex()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

	_, insertErr := orderCollection.InsertOne(ctx, order)
	defer cancel()

	if insertErr != nil {
		return "", insertErr
	}

	return order.Order_id, nil

}
