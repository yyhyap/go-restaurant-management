package controllers

import (
	"context"
	"go-restaurant-management/database"
	"go-restaurant-management/models"
	"log"
	"net/http"
	"strconv"
	"time"

	helper "go-restaurant-management/helpers"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")

func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		//  convert string to int
		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))

		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}

		page, err1 := strconv.Atoi(c.Query("page"))

		if err1 != nil || page < 1 {
			page = 1
		}

		// 0, 10, 20 ...
		startIndex := (page - 1) * recordPerPage

		matchStage := bson.D{
			{Key: "$match", Value: bson.D{{}}},
		}

		groupStage := bson.D{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: bson.D{
					{Key: "_id", Value: "null"},
				}},
				{Key: "total_count", Value: bson.D{
					{Key: "$sum", Value: 1},
				}},
				{Key: "data", Value: bson.D{
					{Key: "$push", Value: "$$ROOT"},
				}},
			}},
		}

		projectStage := bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "total_count", Value: 1},
				{Key: "user_items", Value: bson.D{
					{Key: "$slice", Value: []interface{}{"$data", startIndex, recordPerPage}},
				}},
			}},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, aggregateError := userCollection.Aggregate(ctx, mongo.Pipeline{matchStage, groupStage, projectStage})
		defer cancel()

		if aggregateError != nil {
			log.Println(aggregateError)
			c.JSON(http.StatusBadRequest, gin.H{"error": "error occured while listing user items"})
			return
		}

		var allUsers []bson.M

		if err := result.All(ctx, &allUsers); err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, allUsers[0])
	}
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("user_id")

		var user models.User

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)
		defer cancel()

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		var foundUser models.User

		if err := c.BindJSON(&user); err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		defer cancel()

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
			return
		}

		passwordIsValid, msg := VerifyPassword(*user.Password, *foundUser.Password)

		if !passwordIsValid {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		if foundUser.Email == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
			return
		}

		token, refreshToken, err := helper.GenerateAllTokens(*foundUser.Email, *foundUser.First_name, *foundUser.Last_name, foundUser.User_id)

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while getting token and refreshToken from helper"})
			return
		}

		helper.UpdateAllTokens(token, refreshToken, foundUser.User_id)

		c.JSON(http.StatusOK, foundUser)
	}
}

func Signup() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User

		if err := c.BindJSON(&user); err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validationErr := validate.Struct(user)

		if validationErr != nil {
			log.Println(validationErr)
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		emailCount, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email})
		defer cancel()

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while checking with the email"})
			return
		}

		phoneCount, err := userCollection.CountDocuments(ctx, bson.M{"phone": user.Phone})

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while checking with the phone"})
			return
		}

		// hash password
		password := HashPassword(*user.Password)
		user.Password = &password

		if emailCount > 0 || phoneCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "this email or phone number already exists"})
			return
		}

		user.Created_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing created_at"})
			return
		}

		user.Updated_at, err = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while parsing updated_at"})
			return
		}

		user.ID = primitive.NewObjectID()
		user.User_id = user.ID.Hex()

		token, refreshToken, err := helper.GenerateAllTokens(*user.Email, *user.First_name, *user.Last_name, user.User_id)

		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while getting token and refreshToken from helper"})
			return
		}

		user.Token = &token
		user.Refresh_token = &refreshToken

		resultInsertionNumber, insertErr := userCollection.InsertOne(ctx, user)
		defer cancel()

		if insertErr != nil {
			log.Println(insertErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not created"})
			return
		}

		c.JSON(http.StatusOK, resultInsertionNumber)
	}
}

func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)

	if err != nil {
		log.Panic(err)
	}

	return string(bytes)
}

func VerifyPassword(userPassword string, hashedPassword string) (bool, string) {
	check := true
	msg := ""

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(userPassword))

	if err != nil {
		msg = "email or password is incorrect"
		check = false
	}

	return check, msg
}
