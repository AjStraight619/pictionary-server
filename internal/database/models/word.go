package models

type Word struct {
	Id       uint   `gorm:"primaryKey"`
	Word     string `gorm:"not null"`
	Category string `gorm:"not null"`
}

type JSONWord struct {
	Id       uint   `json:"id"`
	Word     string `json:"word"`
	Category string `json:"category"`
}
