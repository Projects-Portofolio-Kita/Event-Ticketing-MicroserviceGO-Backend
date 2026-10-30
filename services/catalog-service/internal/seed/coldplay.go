package seed

import (
	"fmt"
	"log"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var coldplayEventID = uuid.MustParse("e9b533e4-6725-4c07-ba91-039c4a8a0b01")

type categorySeed struct {
	Code, Name, Subtitle, DotColor, Prefix string
	Price                                  float64
	LeftFrom, LeftTo, RightFrom, RightTo   int
}

var categorySeeds = []categorySeed{
	{
		Code: "cat-2", Name: "CAT 2", Subtitle: "Festival Free Standing",
		DotColor: "#10b981", Prefix: "C2", Price: 2150000,
		LeftFrom: 1, LeftTo: 8, RightFrom: 9, RightTo: 16,
	},
	{
		Code: "cat-3", Name: "CAT 3", Subtitle: "Tribun Bawah",
		DotColor: "#0ea5e9", Prefix: "C3", Price: 1500000,
		LeftFrom: 1, LeftTo: 7, RightFrom: 8, RightTo: 15,
	},
	{
		Code: "cat-4", Name: "CAT 4", Subtitle: "Tribun Atas",
		DotColor: "#475569", Prefix: "C4", Price: 750000,
		LeftFrom: 1, LeftTo: 7, RightFrom: 8, RightTo: 15,
	},
}

var rowNames = []string{"A", "B", "C", "D"}

// Run inserts the Coldplay event, its seat categories and seats on first boot,
// then makes sure every demo event exists (idempotent by code).
func Run(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.Event{}).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		log.Println("[seed] inserting event Coldplay 2026 + kategori + kursi")
		if err := insertColdplay(db); err != nil {
			return err
		}
	}

	return insertDemoEvents(db)
}

func insertColdplay(db *gorm.DB) error {

	event := &model.Event{
		ID:          coldplayEventID,
		Code:        "coldplay-2026",
		Name:        "Coldplay 2026 - Music of the Spheres",
		Venue:       "Stadion Utama Gelora Bung Karno",
		EventDate:   time.Date(2026, 6, 15, 19, 30, 0, 0, time.UTC),
		City:        "Jakarta",
		Category:    model.EventCategoryConcert,
		Description: "Tur dunia Music of the Spheres dengan setlist penuh di GBK Jakarta.",
		SaleMode:    model.SaleModeSeated,
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(event).Error; err != nil {
			return err
		}

		for order, seed := range categorySeeds {
			category := &model.SeatCategory{
				EventID:   event.ID,
				Code:      seed.Code,
				Name:      seed.Name,
				Subtitle:  seed.Subtitle,
				DotColor:  seed.DotColor,
				Price:     seed.Price,
				SortOrder: order + 1,
			}
			if err := tx.Create(category).Error; err != nil {
				return err
			}

			for rowIndex, row := range rowNames {
				for _, number := range rangeOf(seed.LeftFrom, seed.LeftTo) {
					seat := buildSeat(event.ID, category.ID, seed.Prefix, row, number, rowIndex, "left")
					if err := tx.Create(seat).Error; err != nil {
						return err
					}
				}
				for _, number := range rangeOf(seed.RightFrom, seed.RightTo) {
					seat := buildSeat(event.ID, category.ID, seed.Prefix, row, number, rowIndex, "right")
					if err := tx.Create(seat).Error; err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
}

func buildSeat(eventID, categoryID uuid.UUID, prefix, row string, number, rowIndex int, side string) *model.Seat {
	return &model.Seat{
		EventID:    eventID,
		CategoryID: categoryID,
		Label:      fmt.Sprintf("%s-%s%02d", prefix, row, number),
		RowName:    row,
		Number:     number,
		Side:       side,
		SortOrder:  rowIndex*100 + number,
	}
}

func rangeOf(from, to int) []int {
	numbers := make([]int, 0, to-from+1)
	for i := from; i <= to; i++ {
		numbers = append(numbers, i)
	}
	return numbers
}
