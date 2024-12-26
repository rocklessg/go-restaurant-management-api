package models

import(
	"time"
)

type Menu struct {
	BaseEntity					  `bson:",inline"`
	Name       string             `json:"name" validate:"required"`
	Category   string             `json:"category" validate:"required"`
	Start_Date *time.Time         `json:"start_date"`
	End_Date   *time.Time         `json:"end_date"`
	Menu_id    string             `json:"food_id"`
}