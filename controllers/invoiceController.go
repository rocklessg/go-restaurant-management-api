package controller

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"go-restaurant-management/database"
	"go-restaurant-management/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InvoiceViewFormat struct {
	Invoice_id       string
	Payment_method   string
	Order_id         string
	Payment_status   *string
	Payment_due      interface{}
	Table_number     interface{}
	Payment_due_date time.Time
	Order_details    interface{}
}

var invoiceCollection *mongo.Collection = database.OpenCollection(database.Client, "invoice")

func GetInvoices() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure context is canceled to free resources

		// Get pagination parameters from query
		pageStr := c.Query("page")
		limitStr := c.Query("limit")

		// Set default values for page and limit
		page := 1
		limit := 10

		// Parse page and limit, with error handling
		if pageParam, err := strconv.Atoi(pageStr); err == nil && pageParam > 0 {
			page = pageParam
		}
		if limitParam, err := strconv.Atoi(limitStr); err == nil && limitParam > 0 {
			limit = limitParam
		}

		// Calculate skip
		skip := (page - 1) * limit

		// Query invoices with pagination
		cursor, err := invoiceCollection.Find(ctx, bson.M{}, options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while listing invoice items"})
			return
		}
		defer cursor.Close(ctx) // Ensure cursor is closed

		var invoices []bson.M
		if err = cursor.All(ctx, &invoices); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while reading invoice items"})
			log.Fatal(err)
			return
		}

		// Return the paginated invoices
		c.JSON(http.StatusOK, invoices)
	}
}

func GetInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure context is canceled

		invoiceId := c.Query("invoice_id")
		var invoice models.Invoice

		err := invoiceCollection.FindOne(ctx, bson.M{"invoice_id": invoiceId}).Decode(&invoice)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Invoice not found"})
			return // Early return on error
		}

		var invoiceView InvoiceViewFormat
		allOrderItems, err := ItemsByOrder(invoice.Order_id)

		invoiceView.Order_id = invoice.Order_id
		invoiceView.Payment_due_date = invoice.Payment_due_date

		// Handle Payment_method safely
		if invoice.Payment_method != nil {
			invoiceView.Payment_method = *invoice.Payment_method
		} else {
			invoiceView.Payment_method = "null"
		}

		invoiceView.Invoice_id = invoice.Invoice_id
		invoiceView.Payment_status = *&invoice.Payment_status

		// Ensure allOrderItems has elements before accessing
		if len(allOrderItems) > 0 {
			invoiceView.Payment_due = allOrderItems[0]["payment_due"]
			invoiceView.Table_number = allOrderItems[0]["table_number"]
			invoiceView.Order_details = allOrderItems[0]["order_items"]
		} else {
			invoiceView.Payment_due = nil
			invoiceView.Table_number = nil
			invoiceView.Order_details = nil
		}

		c.JSON(http.StatusOK, invoiceView)
	}
}

func CreateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure context is canceled

		var invoice models.Invoice

		// Bind JSON input to the invoice struct
		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if the associated order exists
		var order models.Order
		err := orderCollection.FindOne(ctx, bson.M{"order_id": invoice.Order_id}).Decode(&order)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}

		// Set default payment status if not provided
		if invoice.Payment_status == nil {
			status := "PENDING"
			invoice.Payment_status = &status
		}

		// Set timestamps
		now := time.Now()
		invoice.Payment_due_date = now.AddDate(0, 0, 1) // Due date is 1 day from now
		invoice.Created_at = now
		invoice.Updated_at = now
		invoice.ID = primitive.NewObjectID()
		invoice.Invoice_id = invoice.ID.Hex()

		// Validate the invoice struct
		if validationErr := validate.Struct(invoice); validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Insert the invoice into the collection
		result, insertErr := invoiceCollection.InsertOne(ctx, invoice)
		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invoice could not be created"})
			return
		}

		// Return the result of the insertion
		c.JSON(http.StatusOK, result)
	}
}

func UpdateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Ensure context is canceled

		var invoice models.Invoice
		invoiceId := c.Param("invoice_id")

		// Bind JSON input to the invoice struct
		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		filter := bson.M{"invoice_id": invoiceId}
		updateObj := primitive.D{}

		// Build update object based on provided fields
		if invoice.Payment_method != nil {
			updateObj = append(updateObj, bson.E{"payment_method", invoice.Payment_method})
		}

		if invoice.Payment_status != nil {
			updateObj = append(updateObj, bson.E{"payment_status", invoice.Payment_status})
		}

		// Set updated_at timestamp
		invoice.Updated_at = time.Now()
		updateObj = append(updateObj, bson.E{"updated_at", invoice.Updated_at})

		// Prepare options for the update operation
		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		// Perform the update operation
		result, err := invoiceCollection.UpdateOne(
			ctx,
			filter,
			bson.D{{"$set", updateObj}},
			&opt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invoice update failed"})
			return
		}

		// Return the result of the update operation
		c.JSON(http.StatusOK, result)
	}
}
