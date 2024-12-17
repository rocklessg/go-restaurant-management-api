package models

import(
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BaseEntity struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"` // Use omitempty to skip if not set
    Created_at time.Time          `json:"created_at"`
    Updated_at time.Time          `json:"updated_at"`
}