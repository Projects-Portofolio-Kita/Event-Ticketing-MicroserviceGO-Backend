package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/cors"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/health"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/httperr"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/notify"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/handler"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/provider"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/repository"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/pkg/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// Fail fast: payment owns user data and calls booking-service, so it must
	// never come up without the credentials it needs.
	jwtSecret := authz.RequireEnv("JWT_SECRET")
	serviceToken := authz.RequireEnv("SERVICE_TOKEN")

	db, err := database.NewConnection()
	if err != nil {
		log.Fatal(err)
	}

	if err := db.AutoMigrate(&model.Payment{}); err != nil {
		log.Fatal(err)
	}

	paymentRepository := repository.NewPaymentRepository(db)

	providerMode := strings.ToLower(strings.TrimSpace(os.Getenv("PAYMENT_PROVIDER")))
	var qrisProvider provider.QRISProvider
	if providerMode == "midtrans" {
		qrisProvider = provider.NewMidtransQRISProvider(
			os.Getenv("MIDTRANS_SERVER_KEY"),
			strings.EqualFold(os.Getenv("MIDTRANS_IS_PRODUCTION"), "true"),
			os.Getenv("MIDTRANS_QRIS_ACQUIRER"),
		)
		log.Println("Payment provider: MIDTRANS QRIS")
	} else {
		qrisProvider = provider.NewMockQRISProvider()
		log.Println("Payment provider: MOCK QRIS")
	}

	bookingClient := service.NewHTTPBookingClient()

	paymentService := service.NewPaymentService(
		paymentRepository,
		qrisProvider,
		bookingClient,
		notify.New(),
	)

	paymentHandler := handler.NewPaymentHandler(paymentService)

	router := gin.Default()
	router.Use(cors.New())

	health.Register(router, "payment-service")

	// Unknown routes answer with the standard JSON envelope instead of gin's
	// plain-text 404.
	router.NoRoute(func(c *gin.Context) {
		httperr.JSON(c, http.StatusNotFound, "route tidak ditemukan")
	})

	requireUser := authz.RequireUser(jwtSecret)
	requireUserOrService := authz.RequireUserOrService(jwtSecret, serviceToken)

	api := router.Group("/api/v1")
	{
		// Every payment operation acts on the caller's own account: the
		// service checks the reservation/payment against the token's subject.
		api.POST("/payments", requireUser, paymentHandler.CreatePayment)
		api.GET("/payments/:id", requireUserOrService, paymentHandler.GetPayment)
		api.POST("/payments/:id/confirm", requireUser, paymentHandler.ConfirmPayment)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8084"
	}

	log.Println("Payment Service running on :" + port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
