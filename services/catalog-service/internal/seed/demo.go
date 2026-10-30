package seed

import (
	"fmt"
	"log"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// tierSeed is one quota-based ticket tier (VIP/VVIP/Regular or custom).
type tierSeed struct {
	Name, Subtitle, DotColor string
	Price                    float64
	Quota                    int
}

// eventSeed is one demo event. SEATED events get a generated seat map,
// TIER events only get their ticket tiers (no seats).
type eventSeed struct {
	Code, Name, Venue, City, Category, Description string
	EventDate                                      time.Time
	SaleMode                                       model.SaleMode
	Tiers                                          []tierSeed
	// Seated layout, used only when SaleMode == SEATED.
	Rows, SeatsPerRow, LeftBlock int
}

// demoEvents give the discovery view data across cities, categories and both
// sale modes, so search / location / tier flows are all demonstrable.
var demoEvents = []eventSeed{
	{
		Code: "java-jazz-2027", Name: "Java Jazz Festival 2027",
		Venue: "JIExpo Kemayoran", City: "Jakarta", Category: model.EventCategoryFestival,
		Description: "Festival jazz internasional tiga hari dengan panggung utama dan area outdoor.",
		EventDate:   time.Date(2027, 2, 13, 18, 0, 0, 0, time.UTC),
		SaleMode:    model.SaleModeTier,
		Tiers: []tierSeed{
			{Name: "VVIP", Subtitle: "Panggung depan + akses lounge", DotColor: "#f59e0b", Price: 2500000, Quota: 80},
			{Name: "VIP", Subtitle: "Area premium berdiri", DotColor: "#8b5cf6", Price: 1200000, Quota: 300},
			{Name: "Regular", Subtitle: "General admission", DotColor: "#10b981", Price: 450000, Quota: 1000},
		},
	},
	{
		Code: "tech-summit-2027", Name: "Tech Summit Nusantara 2027",
		Venue: "ICE BSD City", City: "Tangerang", Category: model.EventCategorySeminar,
		Description: "Konferensi teknologi dua hari: keynote, workshop, dan area expo startup.",
		EventDate:   time.Date(2027, 3, 5, 9, 0, 0, 0, time.UTC),
		SaleMode:    model.SaleModeTier,
		Tiers: []tierSeed{
			{Name: "VIP", Subtitle: "Baris depan + makan siang", DotColor: "#8b5cf6", Price: 750000, Quota: 150},
			{Name: "Regular", Subtitle: "Akses sesi utama", DotColor: "#10b981", Price: 350000, Quota: 500},
		},
	},
	{
		Code: "persija-vs-persib-2027", Name: "Persija vs Persib",
		Venue: "Stadion Utama Gelora Bung Karno", City: "Jakarta", Category: model.EventCategorySports,
		Description: "Liga 1 pekan pembuka. Tiket digital, tunjukkan QR saat masuk stadion.",
		EventDate:   time.Date(2027, 4, 11, 15, 30, 0, 0, time.UTC),
		SaleMode:    model.SaleModeTier,
		Tiers: []tierSeed{
			{Name: "VIP", Subtitle: "Tribun tengah beratap", DotColor: "#0ea5e9", Price: 500000, Quota: 200},
			{Name: "Regular", Subtitle: "Tribun utara & selatan", DotColor: "#10b981", Price: 150000, Quota: 2000},
		},
	},
	{
		Code: "ubud-writers-2027", Name: "Ubud Writers & Readers Festival",
		Venue: "Ubud Village Bali", City: "Bali", Category: model.EventCategoryWorkshop,
		Description: "Festival sastra: sesi penulis, diskusi panel, dan lokakarya menulis.",
		EventDate:   time.Date(2027, 5, 20, 10, 0, 0, 0, time.UTC),
		SaleMode:    model.SaleModeTier,
		Tiers: []tierSeed{
			{Name: "Patron", Subtitle: "Semua sesi + dinner penulis", DotColor: "#ec4899", Price: 1500000, Quota: 50},
			{Name: "Regular", Subtitle: "Akses semua sesi harian", DotColor: "#10b981", Price: 350000, Quota: 400},
		},
	},
	{
		Code: "bandung-music-fest-2027", Name: "Bandung Music Fest 2027",
		Venue: "Sabilulungan Bandung", City: "Bandung", Category: model.EventCategoryFestival,
		Description: "Festival musik lintas genre dengan tiga panggung seharian.",
		EventDate:   time.Date(2027, 7, 9, 14, 0, 0, 0, time.UTC),
		SaleMode:    model.SaleModeTier,
		Tiers: []tierSeed{
			{Name: "VVIP", Subtitle: "Front of stage + merch kit", DotColor: "#f59e0b", Price: 1750000, Quota: 60},
			{Name: "VIP", Subtitle: "Area premium", DotColor: "#8b5cf6", Price: 900000, Quota: 250},
			{Name: "Regular", Subtitle: "Festival area", DotColor: "#10b981", Price: 300000, Quota: 1500},
		},
	},
	{
		Code: "surabaya-food-fest-2027", Name: "Surabaya Food & Coffee Fest",
		Venue: "Taman Bungkul", City: "Surabaya", Category: model.EventCategoryFestival,
		Description: "Pesta kuliner dan kopi: 120 tenant, kelas barista, dan panggung akustik.",
		EventDate:   time.Date(2027, 8, 15, 11, 0, 0, 0, time.UTC),
		SaleMode:    model.SaleModeTier,
		Tiers: []tierSeed{
			{Name: "VIP", Subtitle: "Tasting pass 10 tenant", DotColor: "#f97316", Price: 250000, Quota: 300},
			{Name: "Regular", Subtitle: "Akses harian", DotColor: "#10b981", Price: 50000, Quota: 3000},
		},
	},
	{
		Code: "teater-laskar-pelangi-2027", Name: "Teater: Laskar Pelangi",
		Venue: "Graha Bhakti Budaya", City: "Yogyakarta", Category: model.EventCategoryTheater,
		Description: "Pementasan teater musikal karya Andrea Hirata, dua babak.",
		EventDate:   time.Date(2027, 6, 18, 19, 0, 0, 0, time.UTC),
		SaleMode:    model.SaleModeSeated,
		Rows:        3, SeatsPerRow: 10, LeftBlock: 5,
		Tiers: []tierSeed{
			{Name: "CAT 1", Subtitle: "Baris A-B dekat panggung", DotColor: "#f59e0b", Price: 450000},
			{Name: "CAT 2", Subtitle: "Baris C tengah", DotColor: "#0ea5e9", Price: 300000},
		},
	},
}

// insertDemoEvents seeds every demo event whose code is not present yet, so
// re-running the seed (or upgrading an existing database) stays idempotent.
func insertDemoEvents(db *gorm.DB) error {
	for _, seed := range demoEvents {
		var count int64
		if err := db.Model(&model.Event{}).Where("code = ?", seed.Code).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		log.Printf("[seed] inserting event %s (%s, %s)", seed.Code, seed.City, seed.SaleMode)

		if err := db.Transaction(func(tx *gorm.DB) error {
			event := &model.Event{
				ID:          uuid.New(),
				Code:        seed.Code,
				Name:        seed.Name,
				Venue:       seed.Venue,
				EventDate:   seed.EventDate,
				City:        seed.City,
				Category:    seed.Category,
				Description: seed.Description,
				SaleMode:    seed.SaleMode,
				SeatRows:    seed.Rows,
				SeatsPerRow: seed.SeatsPerRow,
			}
			if event.SeatRows == 0 {
				event.SeatRows = defaultSeedRows
			}
			if event.SeatsPerRow == 0 {
				event.SeatsPerRow = defaultSeedSeatsPerRow
			}
			if err := tx.Create(event).Error; err != nil {
				return err
			}

			for order, tier := range seed.Tiers {
				category := &model.SeatCategory{
					EventID:     event.ID,
					Code:        fmt.Sprintf("%s-%d", seed.Code, order+1),
					Name:        tier.Name,
					Subtitle:    tier.Subtitle,
					DotColor:    tier.DotColor,
					Price:       tier.Price,
					SortOrder:   order + 1,
					Quota:       tier.Quota,
					Description: tier.Subtitle,
				}
				if err := tx.Create(category).Error; err != nil {
					return err
				}

				if seed.SaleMode == model.SaleModeSeated {
					seats := generateSeatGrid(event.ID, category.ID, fmt.Sprintf("T%d", order+1), seed.Rows, seed.SeatsPerRow, seed.LeftBlock)
					if err := tx.CreateInBatches(&seats, 250).Error; err != nil {
						return err
					}
				}
			}

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

const (
	defaultSeedRows        = 3
	defaultSeedSeatsPerRow = 10
)

// generateSeatGrid builds a rows x seatsPerRow layout with a left/right split,
// mirroring the Coldplay seed's shape for seated demo events.
func generateSeatGrid(eventID, categoryID uuid.UUID, prefix string, rows, seatsPerRow, leftBlock int) []model.Seat {
	seats := make([]model.Seat, 0, rows*seatsPerRow)

	for rowIndex := 0; rowIndex < rows; rowIndex++ {
		row := rowNames[rowIndex%len(rowNames)]
		for number := 1; number <= seatsPerRow; number++ {
			side := "left"
			if leftBlock > 0 && number > leftBlock {
				side = "right"
			}
			seats = append(seats, model.Seat{
				EventID:    eventID,
				CategoryID: categoryID,
				Label:      fmt.Sprintf("%s-%s%02d", prefix, row, number),
				RowName:    row,
				Number:     number,
				Side:       side,
				SortOrder:  rowIndex*100 + number,
			})
		}
	}

	return seats
}
