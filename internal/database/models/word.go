package models

type Word struct {
	Id       uint   `gorm:"primaryKey" json:"id"`
	Word     string `gorm:"not null" json:"word"`
	Category string `gorm:"not null" json:"category"`
}

type JSONWord struct {
	Id       uint   `json:"id"`
	Word     string `json:"word"`
	Category string `json:"category"`
}
