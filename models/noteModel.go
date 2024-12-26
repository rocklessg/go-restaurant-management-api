package models

import(
)

type Note struct {
	BaseEntity					  `bson:",inline"`
	Text       string             `json:"text"`
	Title      string             `json:"title"`
	Note_id    string             `json:"note_id"`
}