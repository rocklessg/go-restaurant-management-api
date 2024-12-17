package models

import(
)

type Note struct {
	BaseEntity
	Text       string             `json:"text"`
	Title      string             `json:"title"`
	Note_id    string             `json:"note_id"`
}