package models

import(
	"time"
)

type Order struct {
	BaseEntity					  `bson:",inline"`		// Embeded base entity
	Order_Date time.Time          `json:"order_date" validate:"required"`
	Order_id   string             `json:"order_id"`
	Table_id   *string            `json:"table_id" validate:"required"`
}