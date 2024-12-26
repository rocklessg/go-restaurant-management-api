package controller

import(
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	"go-restaurant-management/database"
	"go-restaurant-management/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var foodCollection *mongo.Collection = database.OpenCollection(database.Client, "food")
var validate = validator.New()

func GetFoods() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure context is canceled to free resources

		// Retrieve query parameters with defaults
		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}

		page, pageErr := strconv.Atoi(c.Query("page"))
		if pageErr != nil || page < 1 {
			page = 1
		}

		startIndex := (page - 1) * recordPerPage

		// Define aggregation stages
		matchStage := bson.D{{Key: "$match", Value: bson.D{}}}
		groupStage := bson.D{
			{"$group", bson.D{
				{"_id", nil}, // Group all documents together
				{"total_count", bson.D{{"$sum", 1}}},
				{"data", bson.D{{"$push", "$$ROOT"}}},
			}},
		}
		projectStage := bson.D{
			{"$project", bson.D{
				{"_id", 0},
				{"total_count", 1},
				{"food_items", bson.D{{"$slice", []interface{}{"$data", startIndex, recordPerPage}}}},
			}},
		}

		// Execute the aggregation
		result, err := foodCollection.Aggregate(ctx, mongo.Pipeline{
			matchStage, groupStage, projectStage,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing food items"})
			return // Exit the handler if there's an error
		}

		// Create a value into which the single document can be decoded
		var allFoods []bson.M
		if err = result.All(ctx, &allFoods); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while processing results"})
			return // Exit the handler if there's an error
		}

		if len(allFoods) == 0 {
			c.JSON(http.StatusOK, []bson.M{}) // Return an empty list if no items are found
			return
		}
		c.JSON(http.StatusOK, allFoods)
	}
}

func GetFood() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var food models.Food

		err := foodCollection.FindOne(ctx, bson.M{"food_id": c.Query("food_id")}).Decode(&food)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while fetching the food item"})
			return
		}
		c.JSON(http.StatusOK, food)
	}
}

func CreateFood() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure context is canceled to free resources

		var menu models.Menu
		var food models.Food

		// Bind JSON to food struct
		if err := c.BindJSON(&food); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate the food struct
		if validationErr := validate.Struct(food); validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Check if the menu exists
		if err := menuCollection.FindOne(ctx, bson.M{"menu_id": food.Menu_id}).Decode(&menu); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "menu not found"})
			return
		}

		// Set timestamps and ID for the food item
		food.ID = primitive.NewObjectID()
		food.Food_id = food.ID.Hex()
		now := time.Now()
		food.Created_at = now
		food.Updated_at = now

		// Format price to two decimal places
		num := toFixed(*food.Price, 2)
		food.Price = &num

		// Insert the food item into the database
		result, insertErr := foodCollection.InsertOne(ctx, food)
		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Food item was not created"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func UpdateFood() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure context is canceled to free resources

		var menu models.Menu
		var food models.Food

		foodId := c.Param("food_id")

		// Bind JSON to food struct
		if err := c.BindJSON(&food); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var updateObj primitive.D

		// Build update object based on non-nil fields
		if food.Name != nil {
			updateObj = append(updateObj, bson.E{"name", food.Name})
		}
		if food.Price != nil {
			updateObj = append(updateObj, bson.E{"price", food.Price})
		}
		if food.Food_image != nil {
			updateObj = append(updateObj, bson.E{"food_image", food.Food_image})
		}

		// Check if the menu exists
		if food.Menu_id != nil {
			if err := menuCollection.FindOne(ctx, bson.M{"menu_id": food.Menu_id}).Decode(&menu); err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Menu was not found"})
				return
			}
			updateObj = append(updateObj, bson.E{"menu", food.Menu_id})
		}

		// Set updated_at timestamp directly
		food.Updated_at = time.Now()
		updateObj = append(updateObj, bson.E{"updated_at", food.Updated_at})

		// Define filter and options for update
		filter := bson.M{"food_id": foodId}
		upsert := true
		opt := options.Update().SetUpsert(upsert)

		// Perform the update operation
		result, err := foodCollection.UpdateOne(ctx, filter, bson.D{{"$set", updateObj}}, opt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Food item update failed"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}


func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}