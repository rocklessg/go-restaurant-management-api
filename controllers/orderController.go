package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go-restaurant-management/database"
	"go-restaurant-management/models"

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
		defer cancel() // Ensure context is canceled

		// Retrieve all orders from the collection
		result, err := orderCollection.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while listing order items"})
			return
		}

		// Collect all orders into a slice
		var allOrders []bson.M
		if err := result.All(ctx, &allOrders); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while processing order items"})
			return
		}

		// Return the list of orders
		c.JSON(http.StatusOK, allOrders)
	}
}

func GetOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure context is canceled

		orderId := c.Param("order_id")
		var order models.Order

		// Attempt to find the order by order_id
		err := orderCollection.FindOne(ctx, bson.M{"order_id": orderId}).Decode(&order)
		if err != nil {
			// Check if the error is due to no documents found
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching the order"})
			return
		}

		// Return the found order
		c.JSON(http.StatusOK, order)
	}
}

func CreateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure context is canceled

		var order models.Order

		// Bind JSON input to the order struct
		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate the order struct
		if validationErr := validate.Struct(order); validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Check if the table exists if Table_id is provided
		if order.Table_id != nil {
			var table models.Table
			if err := tableCollection.FindOne(ctx, bson.M{"table_id": order.Table_id}).Decode(&table); err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
				return
			}
		}

		// Set created and updated timestamps
		order.Created_at = time.Now()
		order.Updated_at = time.Now()

		// Generate a new ObjectID and set the order ID
		order.ID = primitive.NewObjectID()
		order.Order_id = order.ID.Hex()

		// Insert the order into the collection
		result, insertErr := orderCollection.InsertOne(ctx, order)
		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create Order item."})
			return
		}

		// Return the result of the insertion
		c.JSON(http.StatusOK, result)
	}
}

func UpdateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure context is canceled

		var order models.Order
		var updateObj primitive.D

		// Extract order ID from the request parameters
		orderId := c.Param("order_id")
		if orderId == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "order_id is required"})
			return
		}

		// Bind JSON body to the order model
		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check and validate table_id
		if order.Table_id != nil {
			var table models.Table
			err := menuCollection.FindOne(ctx, bson.M{"table_id": order.Table_id}).Decode(&table)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
				return
			}
			updateObj = append(updateObj, bson.E{Key: "table_id", Value: order.Table_id})
		}

		// Update the timestamp
		order.Updated_at = time.Now()
		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: order.Updated_at})

		// Prepare the filter and update options
		filter := bson.M{"order_id": orderId}
		opts := options.Update().SetUpsert(true)

		// Perform the update operation
		result, err := orderCollection.UpdateOne(ctx, filter, bson.D{{Key: "$set", Value: updateObj}}, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Order update failed"})
			return
		}

		// Return the update result
		c.JSON(http.StatusOK, gin.H{"result": result})
	}
}

func OrderItemOrderCreator(order models.Order) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel() // Ensure context is canceled

	// Set created and updated timestamps
	now := time.Now()
	order.Created_at = now
	order.Updated_at = now

	// Generate a new ObjectID and set the order ID
	order.ID = primitive.NewObjectID()
	order.Order_id = order.ID.Hex()

	// Insert the order into the collection
	_, err := orderCollection.InsertOne(ctx, order)
	if err != nil {
		return "", fmt.Errorf("failed to insert order: %w", err)
	}
	return order.Order_id, nil
}
