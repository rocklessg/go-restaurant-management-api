package controller

import (
	"context"
	"errors"
	"net/http"
	"time"
	"fmt"

	"go-restaurant-management/database"
	"go-restaurant-management/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var tableCollection *mongo.Collection = database.OpenCollection(database.Client, "table")

func GetTables() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		result, err := tableCollection.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while listing table items"})
			return
		}

		var allTables []bson.M
		if err := result.All(ctx, &allTables); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while retrieving table items"})
			return
		}
		c.JSON(http.StatusOK, allTables)
	}
}

func GetTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		tableId := c.Query("table_id")
		var table models.Table

		fmt.Printf("Searching for table with ID: %s", tableId)
		err := tableCollection.FindOne(ctx, bson.M{"table_id": tableId}).Decode(&table)
		if err != nil {
			fmt.Printf("Error occurred: %v", err) // Log the error
			// Check if the error is due to no documents found
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusNotFound, gin.H{"error": "table not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while fetching the tables"})
			return
		}
		fmt.Printf("Fetched table: %+v", table) // Log the fetched table
		c.JSON(http.StatusOK, table)
	}
}

func CreateTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure context is canceled

		var table models.Table
		if err := c.BindJSON(&table); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate table structure
		if validationErr := validate.Struct(table); validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Set timestamps
		table.Created_at = time.Now()
		table.Updated_at = time.Now()

		// Generate ID for the new table
		table.ID = primitive.NewObjectID()
		table.Table_id = table.ID.Hex()

		// Insert the table into the database
		result, insertErr := tableCollection.InsertOne(ctx, table)
		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create Table item"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func UpdateTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure context is canceled

		var table models.Table
		tableId := c.Param("table_id")

		if err := c.BindJSON(&table); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Prepare update object
		updateObj := primitive.D{}
		if table.Number_of_guests != nil {
			updateObj = append(updateObj, bson.E{"number_of_guests", table.Number_of_guests})
		}
		if table.Table_number != nil {
			updateObj = append(updateObj, bson.E{"table_number", table.Table_number})
		}

		// Update the timestamp
		table.Updated_at = time.Now()
		updateObj = append(updateObj, bson.E{"updated_at", table.Updated_at})

		// Set upsert option
		opt := options.Update().SetUpsert(true)

		// Execute the update
		filter := bson.M{"table_id": tableId}

		result, err := tableCollection.UpdateOne(ctx, filter, bson.D{{"$set", updateObj}}, opt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Table item update failed"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
