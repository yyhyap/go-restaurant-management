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

type InvoiceViewFormat struct {
	Invoice_id       string
	Payment_method   string
	Payment_status   *string
	Payment_due      interface{}
	Payment_due_date time.Time
	Table_number     interface{}
	Order_id         string
	Order_details    interface{}
}

var invoiceCollection *mongo.Collection = database.OpenCollection(database.Client, "invoice")

func GetInvoices() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, err := invoiceCollection.Find(ctx, bson.M{})
		defer cancel()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while retrieving invoices from database"})
			return
		}

		var allInvoices []bson.M

		if err := result.All(ctx, &allInvoices); err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, allInvoices)
	}
}

func GetInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		invoiceId := c.Param("invoice_id")
		var invoice models.Invoice

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		err := invoiceCollection.FindOne(ctx, bson.M{"invoice_id": invoiceId}).Decode(&invoice)
		defer cancel()

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invoice not found"})
			return
		}

		var invoiceView InvoiceViewFormat

		allOrderItems, err := ItemsByOrder(invoice.Invoice_id)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		invoiceView.Order_id = invoice.Order_id
		invoiceView.Invoice_id = invoice.Invoice_id
		invoiceView.Payment_due_date = invoice.Payment_due_date

		invoiceView.Payment_method = "null"
		if invoice.Payment_method != nil {
			invoiceView.Payment_method = *invoice.Payment_method
		}

		invoiceView.Payment_status = invoice.Payment_status
		invoiceView.Payment_due = allOrderItems[0]["payment_due"]
		invoiceView.Table_number = allOrderItems[0]["table_number"]
		invoiceView.Order_details = allOrderItems[0]["order_items"]

		c.JSON(http.StatusOK, invoiceView)
	}
}

func CreateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		var invoice models.Invoice

		if err := c.BindJSON(&invoice); err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var order models.Order

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		err := orderCollection.FindOne(ctx, bson.M{"order_id": invoice.Order_id}).Decode(&order)
		defer cancel()

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "order not found"})
			return
		}

		// AddDate(years, months, days)
		invoice.Payment_due_date, err = time.Parse(time.RFC3339, time.Now().AddDate(0, 0, 1).Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing payment_due_date"})
			return
		}

		invoice.Created_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing created_at"})
			return
		}

		invoice.Updated_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing updated_at"})
			return
		}

		invoice.ID = primitive.NewObjectID()
		invoice.Invoice_id = invoice.ID.Hex()

		validationError := validate.Struct(invoice)

		if validationError != nil {
			log.Println(validationError)
			c.JSON(http.StatusBadRequest, gin.H{"error": validationError.Error()})
			return
		}

		result, insertErr := invoiceCollection.InsertOne(ctx, invoice)
		defer cancel()

		if insertErr != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invoice is not created due to some errors"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func UpdateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		var invoice models.Invoice

		if err := c.BindJSON(&invoice); err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		invoiceId := c.Param("invoice_id")
		var updateObj primitive.D

		if invoice.Payment_method != nil {
			updateObj = append(updateObj, bson.E{Key: "payment_method", Value: invoice.Payment_method})
		}

		if invoice.Payment_status != nil {
			updateObj = append(updateObj, bson.E{Key: "payment_status", Value: invoice.Payment_status})
		}

		// // if payment status is nil, update the status to "PENDING"
		// status := "PENDING"
		// if invoice.Payment_status == nil {
		// 	invoice.Payment_status = &status
		// 	updateObj = append(updateObj, bson.E{Key: "payment_status", Value: invoice.Payment_status})
		// }

		var err error
		invoice.Updated_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing updated_at"})
			return
		}

		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: invoice.Updated_at})

		upsert := true
		filter := bson.M{"invoice_id": invoiceId}
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, err := invoiceCollection.UpdateOne(
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invoice updated failed"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
