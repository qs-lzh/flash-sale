package workflow

import (
	"encoding/json"
	"log"

	"github.com/qs-lzh/flash-sale/internal/mq"
	"github.com/qs-lzh/flash-sale/internal/service/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PaymentWorkflow struct {
	paymentService domain.PaymentService
	mqConn         *amqp.Connection
}

func NewPaymentWorkflow(paymentService domain.PaymentService, mqConn *amqp.Connection) *PaymentWorkflow {
	return &PaymentWorkflow{
		paymentService: paymentService,
		mqConn:         mqConn,
	}
}

func (w *PaymentWorkflow) Start(mqConn *amqp.Connection) error {
	if err := w.ConsumePaymentCreate(mqConn); err != nil {
		return err
	}
	if err := w.ConsumePaymentTimeout(mqConn); err != nil {
		return err
	}

	return nil
}

func (w *PaymentWorkflow) ConsumePaymentCreate(conn *amqp.Connection) error {
	ch, err := mq.NewChannel(conn)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(mq.ReservationToPaymentImmediateQueue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			go func() {
				reservationID, err := w.handlePaymentMessage(msg)
				if err != nil {
					log.Printf("Failed to handle payment message: %v", err)
				} else {
					// no error means payment success,
					// so send message to tell db to create order
					if err := mq.SendImmediateMessage(ch, mq.PaymentToOrderImmediateQueue,
						mq.PaymentToOrderImmediateMessage{
							ReservationID: reservationID,
						}); err != nil {
						log.Printf("Failed to send message: %v", err)
					}
				}
			}()
		}
	}()

	return nil
}

func (w *PaymentWorkflow) handlePaymentMessage(msg amqp.Delivery) (reservationID uint, err error) {
	var message mq.ReservationToPaymentImmediateMessage
	if err := json.Unmarshal(msg.Body, &message); err != nil {
		msg.Nack(false, false)
		return 0, err
	}

	if err := w.paymentService.StartMockPay(message.ReservationID); err != nil {
		msg.Nack(false, true)
		return 0, err
	}

	msg.Ack(false)

	return message.ReservationID, nil
}

func (w *PaymentWorkflow) ConsumePaymentTimeout(mqConn *amqp.Connection) error {
	ch, err := mq.NewChannel(mqConn)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(mq.ReservationToPaymentTimeoutQueue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			w.handlePaymentTimeout(msg)
		}
	}()

	return nil
}

func (w *PaymentWorkflow) handlePaymentTimeout(msg amqp.Delivery) {
	var message mq.ReservationToPaymentDelayMessage
	if err := json.Unmarshal(msg.Body, &message); err != nil {
		msg.Nack(false, false)
		return
	}
	if err := w.paymentService.MarkTimeout(message.ReservationID); err != nil {
		msg.Nack(false, true)
		return
	}

	msg.Ack(false)
}
