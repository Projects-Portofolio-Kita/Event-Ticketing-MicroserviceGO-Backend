// Package database provides PostgreSQL persistence and initial seeding.
//
// File: seed.go
// Usability: Idempotently seeds initial venue data records (GBK, ICE BSD, JIExpo) with conflict resolution.
package database

import (
	"fmt"
	"log"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SeedVenues idempotently seeds the 3 venues (Stadion Utama Gelora Bung Karno, ICE BSD, JIExpo Kemayoran).
func SeedVenues(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db connection is required")
	}
	venues := []model.Venue{
		{
			ID: VenueIDGBK, Name: "Stadion Utama Gelora Bung Karno",
			Address: "Jl. Pintu Satu Senayan, Gelora, Kecamatan Tanah Abang", City: "Jakarta Pusat",
			TotalCapacity: 77193, LayoutConfig: `{"type":"stadium","sections":["CAT1","CAT2","VIP","VVIP"]}`,
		},
		{
			ID: VenueIDICEBSD, Name: "ICE BSD (Indonesia Convention Exhibition)",
			Address: "Jl. BSD Grand Boulevard No.1, Pagedangan", City: "Tangerang",
			TotalCapacity: 10000, LayoutConfig: `{"type":"convention_hall","sections":["Hall 1","Hall 2","Hall 3"]}`,
		},
		{
			ID: VenueIDJIExpo, Name: "JIExpo Kemayoran",
			Address: "Gedung Pusat Niaga, Arena JIEXPO Kemayoran", City: "Jakarta Pusat",
			TotalCapacity: 12000, LayoutConfig: `{"type":"expo_center","sections":["Hall A","Hall B","Hall C","Hall D"]}`,
		},
	}
	if err := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoNothing: true}).Create(&venues).Error; err != nil {
		return fmt.Errorf("failed to seed venues: %w", err)
	}
	log.Printf("[INFO] Successfully seeded %d venues", len(venues))
	return nil
}
