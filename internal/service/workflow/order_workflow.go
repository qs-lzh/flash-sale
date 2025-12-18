package workflow

import (
	"encoding/json"
	"log"

	"github.com/qs-lzh/flash-sale/internal/cache"
	"github.com/qs-lzh/flash-sale/internal/mq"
	"github.com/qs-lzh/flash-sale/internal/service/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

type OrderWorkflow struct {
	cache        *cache.RedisCache
	orderService domain.OrderService
}

func NewOrderWorkflow(cache *cache.RedisCache, orderService domain.OrderService) *OrderWorkflow {
	return &OrderWorkflow{
		cache:        cache,
		orderService: orderService,
	}
}

func (w *OrderWorkflow) Start(mqConn *amqp.Connection) error {
	if err := w.ConsumeOrderCreation(mqConn); err != nil {
		return err
	}
	return nil
}

func (w *OrderWorkflow) ConsumeOrderCreation(conn *amqp.Connection) error {
	ch, err := mq.NewChannel(conn)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(mq.PaymentToOrderImmediateQueue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			if err := w.handleOrderCreation(msg); err != nil {
				log.Printf("Failed to handle order creation: %v", err)
			}
		}
	}()

	return nil
}

func (w *OrderWorkflow) handleOrderCreation(msg amqp.Delivery) error {
	var message mq.PaymentToOrderImmediateMessage
	if err := json.Unmarshal(msg.Body, &message); err != nil {
		msg.Nack(false, false)
		return err
	}

	if err := w.orderService.CreateOrderFromReservation(message.ReservationID); err != nil {
		msg.Nack(false, true)
		return err
	}

	msg.Ack(false)

	return nil
}
