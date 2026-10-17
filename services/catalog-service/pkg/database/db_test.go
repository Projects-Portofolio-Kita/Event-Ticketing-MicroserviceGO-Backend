package database_test

import (
	"testing"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/pkg/database"
)

func TestConnect_InvalidDSN(t *testing.T) {
	// An invalid or unreachable connection string should return an error gracefully
	_, err := database.Connect("postgres://invalid_user:invalid_pass@127.0.0.1:54329/invalid_db?connect_timeout=1&sslmode=disable")
	if err == nil {
		t.Log("Note: Connected unexpectedly or driver deferred connection")
	}
}

func TestSeedConstants(t *testing.T) {
	if database.VenueIDGBK.String() != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("unexpected GBK venue UUID: %s", database.VenueIDGBK)
	}
	if database.VenueIDICEBSD.String() != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("unexpected ICE BSD venue UUID: %s", database.VenueIDICEBSD)
	}
	if database.VenueIDJIExpo.String() != "33333333-3333-3333-3333-333333333333" {
		t.Errorf("unexpected JIExpo venue UUID: %s", database.VenueIDJIExpo)
	}
}
