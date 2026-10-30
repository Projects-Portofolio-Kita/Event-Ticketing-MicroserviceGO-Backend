package main

import (
	"log"
	"os"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/gateway/internal/gateway"
)

func main() {
	g := gateway.New()

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Gateway running on :" + port)

	if err := g.Engine().Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
