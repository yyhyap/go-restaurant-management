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

type OrderItemPack struct {
	Table_id    *string
	Order_items []models.OrderItem
}

var orderItemCollection *mongo.Collection = database.OpenCollection(database.Client, "orderItem")

func GetOrderItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, err := orderItemCollection.Find(ctx, bson.M{})
		defer cancel()

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while retrieving order items from database"})
			return
		}

		var allOrderItems []bson.M

		if err := result.All(ctx, &allOrderItems); err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, allOrderItems)
	}
}

func GetOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderItemId := c.Param("orderItem_id")

		var orderItem models.OrderItem

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		err := orderItemCollection.FindOne(ctx, bson.M{"order_item_id": orderItemId}).Decode(&orderItem)
		defer cancel()

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "order item not found"})
			return
		}

		c.JSON(http.StatusOK, orderItem)
	}
}

func GetOrderItemsByOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Param("order_id")

		allOrderItems, err := ItemsByOrder(orderId)

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while retrieving order items by order from database"})
			return
		}

		c.JSON(http.StatusOK, allOrderItems)
	}
}

func ItemsByOrder(id string) (OrderItems []primitive.M, err error) {
	matchStage := bson.D{
		{Key: "$match", Value: bson.D{
			{Key: "order_id", Value: id},
		}},
	}

	// https://docs.mongodb.com/manual/reference/operator/aggregation/lookup/
	lookupFoodStage := bson.D{
		{Key: "$lookup", Value: bson.D{
			// from: <collection to join>,
			{Key: "from", Value: "food"},
			// localField: <field from the input documents>,
			// food_id in orderItemCollection
			{Key: "localField", Value: "food_id"},
			// foreignField: <field from the documents of the "from" collection>,
			// food_id in foodCollection
			{Key: "foreignField", Value: "food_id"},
			// as: <output array field>
			{Key: "as", Value: "food"},
		}},
	}

	// https://docs.mongodb.com/manual/reference/operator/aggregation/unwind/
	// Deconstructs an array field from the input documents to output a document for each element.
	// Each output document is the input document with the value of the array field replaced by the element.
	// for example, lookupFoodStage provide a output array field named as 'food'
	// use $unwind to generate document (row in SQL database) for each of the elements in 'food' array
	unwindFoodStage := bson.D{
		{Key: "$unwind", Value: bson.D{
			// from the array 'food'
			{Key: "path", Value: "$food"},
			// If true, if the path is null, missing, or an empty array, $unwind outputs the document.
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}},
	}

	lookupOrderStage := bson.D{
		{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "order"},
			{Key: "localField", Value: "order_id"},
			// order_id from orderCollection
			{Key: "foreignField", Value: "order_id"},
			{Key: "as", Value: "order"},
		}},
	}

	unwindOrderStage := bson.D{
		{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$order"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}},
	}

	lookupTableStage := bson.D{
		{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "table"},
			// order.table_id >>> unwinded the 'order' array and generate document for each of the elements in the array
			{Key: "localField", Value: "order.table_id"},
			// table_id from tableCollection
			{Key: "foreignField", Value: "table_id"},
			{Key: "as", Value: "table"},
		}},
	}

	unwindTableStage := bson.D{
		{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$table"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}},
	}

	projectStage := bson.D{
		{Key: "$project", Value: bson.D{
			{Key: "id", Value: 0},
			{Key: "amount", Value: "$food.price"},
			{Key: "total_count", Value: 1},
			{Key: "food_name", Value: "$food.name"},
			{Key: "food_image", Value: "$food.food_image"},
			{Key: "table_number", Value: "$table.table_number"},
			{Key: "table_id", Value: "$table.table_id"},
			{Key: "order_id", Value: "$order.order_id"},
			{Key: "price", Value: "$food.price"},
			{Key: "quantity", Value: 1},
		}},
	}

	groupStage := bson.D{
		{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "order_id", Value: "$order_id"},
				{Key: "table_id", Value: "$table_id"},
				{Key: "table_number", Value: "$table_number"},
			}},
			{Key: "payment_due", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
			{Key: "total_count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "order_items", Value: bson.D{{Key: "$push", Value: "$$ROOT"}}},
		}},
	}

	projectStage2 := bson.D{
		{Key: "$project", Value: bson.D{
			{Key: "id", Value: 0},
			{Key: "payment_due", Value: 1},
			{Key: "total_count", Value: 1},
			{Key: "table_number", Value: "$_id.table_number"},
			{Key: "order_items", Value: 1},
		}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

	result, err := orderItemCollection.Aggregate(ctx, mongo.Pipeline{
		matchStage,
		lookupFoodStage,
		unwindFoodStage,
		lookupOrderStage,
		unwindOrderStage,
		lookupTableStage,
		unwindTableStage,
		projectStage,
		groupStage,
		projectStage2,
	})
	defer cancel()

	if err != nil {
		log.Println(err)
		return []primitive.M{{"error": err.Error()}}, err
	}

	if err := result.All(ctx, &OrderItems); err != nil {
		log.Println(err)
		return []primitive.M{{"error": err.Error()}}, err
	}

	return OrderItems, err
}

func CreateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		var order models.Order
		var orderItemPack OrderItemPack

		if err := c.BindJSON(&orderItemPack); err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error})
			return
		}

		var err error

		order.Order_date, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing order_date"})
			return
		}

		order.Table_id = orderItemPack.Table_id

		// create an order whenever need to create an order item
		order_id, err := OrderItemOrderCreator(order)

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		orderItemsToBeInserted := []interface{}{}

		// inserting order items for each order
		for _, orderItem := range orderItemPack.Order_items {
			orderItem.Order_id = order_id

			validationError := validate.Struct(orderItem)

			if validationError != nil {
				log.Println(validationError)
				c.JSON(http.StatusBadRequest, gin.H{"error": validationError.Error()})
				return
			}

			orderItem.ID = primitive.NewObjectID()
			orderItem.Order_item_id = orderItem.ID.Hex()

			orderItem.Created_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

			if err != nil {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing created_at"})
				return
			}

			orderItem.Updated_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

			if err != nil {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing updated_at"})
				return
			}

			var num = toFixed(*orderItem.Unit_price, 2)
			orderItem.Unit_price = &num

			// insert the order item into an array
			orderItemsToBeInserted = append(orderItemsToBeInserted, orderItem)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		insertOrderItemsResult, insertError := orderItemCollection.InsertMany(ctx, orderItemsToBeInserted)
		defer cancel()

		if insertError != nil {
			log.Println(insertError)
			c.JSON(http.StatusInternalServerError, gin.H{"error": insertError.Error()})
			return
		}

		c.JSON(http.StatusOK, insertOrderItemsResult)
	}
}

func UpdateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderItemId := c.Param("orderItem_id")
		var orderItem models.OrderItem

		if err := c.BindJSON(&orderItem); err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error})
			return
		}

		var updateObj primitive.D

		if orderItem.Unit_price != nil {
			updateObj = append(updateObj, bson.E{Key: "unit_price", Value: orderItem.Unit_price})
		}

		if orderItem.Quantity != nil {
			updateObj = append(updateObj, bson.E{Key: "quantity", Value: orderItem.Quantity})
		}

		if orderItem.Food_id != nil {
			updateObj = append(updateObj, bson.E{Key: "food_id", Value: orderItem.Food_id})
		}

		var err error
		orderItem.Updated_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing updated_at"})
			return
		}

		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: orderItem.Updated_at})

		upsert := true
		filter := bson.M{"order_item_id": orderItemId}
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, err := orderItemCollection.UpdateOne(
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "order item updated failed"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
