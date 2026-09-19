package repository

import (
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(payment *model.Payment) error {
	return r.db.Create(payment).Error
}

func (r *PaymentRepository) FindByID(id uuid.UUID) (*model.Payment, error) {
	var payment model.Payment

	err := r.db.First(&payment, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) FindByReservationID(reservationID uuid.UUID) (*model.Payment, error) {
	var payment model.Payment

	err := r.db.
		Where("reservation_id = ?", reservationID).
		First(&payment).Error

	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) Update(payment *model.Payment) error {
	return r.db.Save(payment).Error
}