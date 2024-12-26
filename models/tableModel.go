package models

import(
)

type Table struct {
	BaseEntity							`bson:",inline"`
	Number_of_guests *int               `json:"number_of_guests" validate:"required"`
	Table_number     *int               `json:"table_number" validate:"required"`
	Table_id         string             `json:"table_id"`
}