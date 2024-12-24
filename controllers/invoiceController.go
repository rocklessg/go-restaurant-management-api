package controller

import(
    "context"
    "fmt"
    "net/http"
    "log"
    "time"
    "strconv"

    "go-restaurant-management/models"
    "go-restaurant-management/database"

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

        invoiceId := c.Param("invoice_id")
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
