package helpers

import (
	"context"
	"go-restaurant-management/database"
	"log"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")

var SECRET_KEY string = GetEnvVariable("SECRET_KEY")

type SignedDetails struct {
	Email      string
	First_name string
	Last_name  string
	Uid        string
	jwt.StandardClaims
}

func GenerateAllTokens(email string, firstName string, lastName string, uid string) (signedToken string, signedRefreshToken string, err error) {

	if SECRET_KEY == "" {
		SECRET_KEY = "test"
	}

	claims := &SignedDetails{
		Email:      email,
		First_name: firstName,
		Last_name:  lastName,
		Uid:        uid,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(24)).Unix(),
		},
	}

	refreshClaims := &SignedDetails{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(168)).Unix(),
		},
	}

	signedToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(SECRET_KEY))

	if err != nil {
		log.Panic(err)
		return
	}

	signedRefreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(SECRET_KEY))

	if err != nil {
		log.Panic(err)
		return
	}

	return signedToken, signedRefreshToken, err
}

func UpdateAllTokens(signedToken, signedRefreshToken, userId string) {
	var updateObj primitive.D

	updateObj = append(updateObj, bson.E{Key: "token", Value: signedToken})
	updateObj = append(updateObj, bson.E{Key: "refreshToken", Value: signedRefreshToken})

	Updated_at, err := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

	if err != nil {
		log.Println(err)
		return
	}

	updateObj = append(updateObj, bson.E{Key: "updated_at", Value: Updated_at})

	upsert := true
	filter := bson.M{"user_id": userId}
	opt := options.UpdateOptions{
		Upsert: &upsert,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*100)

	_, err = userCollection.UpdateOne(
		ctx,
		filter,
		bson.D{
			bson.E{Key: "$set", Value: updateObj},
		},
		&opt,
	)
	defer cancel()

	if err != nil {
		log.Panic(err)
		return
	}
}

func ValidateToken(signedToken string) (claims *SignedDetails, msg string) {
	if SECRET_KEY == "" {
		SECRET_KEY = "test"
	}

	token, err := jwt.ParseWithClaims(
		signedToken,
		&SignedDetails{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(SECRET_KEY), nil
		},
	)

	if err != nil {
		msg = err.Error()
		return
	}

	// type assesrtion to type '&SignedDetails'
	claims, ok := token.Claims.(*SignedDetails)

	if !ok {
		msg = "invalid token"
		return
	}

	if claims.ExpiresAt < time.Now().Local().Unix() {
		msg = "token has expired"
		return
	}

	return claims, msg
}
