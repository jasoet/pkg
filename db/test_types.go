//go:build integration

package db

import "time"

// Define test models that match the schema in default.sql
type Product struct {
	ID            string `gorm:"primaryKey;type:uuid"`
	Name          string `gorm:"not null"`
	Description   string
	Category      string  `gorm:"not null"`
	Price         float64 `gorm:"not null"`
	StockQuantity int     `gorm:"not null"`
	WeightKg      float64
	Dimensions    string
	IsAvailable   bool      `gorm:"default:true"`
	CreatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time
}

type Customer struct {
	ID               string `gorm:"primaryKey;type:uuid"`
	FirstName        string `gorm:"not null"`
	LastName         string `gorm:"not null"`
	Email            string `gorm:"unique;not null"`
	Phone            string
	Address          string
	City             string
	Country          string
	PostalCode       string
	IsActive         bool      `gorm:"default:true"`
	LoyaltyPoints    int       `gorm:"default:0"`
	RegistrationDate time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}
