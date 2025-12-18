package workflow

import (
	"github.com/qs-lzh/flash-sale/internal/mq"
	"github.com/qs-lzh/flash-sale/internal/service/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ReservationWorkflow struct {
	ReservationService domain.ReservationService
	MQConn             *amqp.Connection
}

func NewReservationWorkflow(reservationService domain.ReservationService, mqConn *amqp.Connection) *ReservationWorkflow {
	return &ReservationWorkflow{
		ReservationService: reservationService,
		MQConn:             mqConn,
	}
}

func (w *ReservationWorkflow) Reserve(userID, showtimeID uint) error {
	reservationID, err := w.ReservationService.Reserve(userID, showtimeID)
	if err != nil {
		return err
	}

	ch, err := mq.NewChannel(w.MQConn)
	if err != nil {
		return err
	}

	if err := mq.SendImmediateMessage(ch, mq.ReservationToPaymentImmediateQueue,
		mq.ReservationToPaymentImmediateMessage{
			ReservationID: reservationID,
			Price:         1,
		}); err != nil {
		return err
	}

	if err := mq.SendTimeoutMessage(ch, mq.ReservationToPaymentDelayQueue,
		mq.ReservationToPaymentDelayMessage{
			ReservationID: reservationID,
		}); err != nil {
		return err
	}

	return nil
}
