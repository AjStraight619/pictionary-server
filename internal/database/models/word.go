package models

type Word struct {
	Id       uint   `gorm:"primaryKey"`
	Word     string `gorm:"not null"`
	Category string `gorm:"not null"`
}
