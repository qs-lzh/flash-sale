package mq

// Queue names and message definitions

// immediate queue from reservation to payment service
// deliver message to notify payment service to handle a payment of the reservation
const (
	ReservationToPaymentImmediateQueue = "reservation.payment.pay.immediate"
)

type ReservationToPaymentImmediateMessage struct {
	ReservationID uint `json:"reservation_id"`
	Price         int  `json:"price"`
}

// delay queue from reservation to payment service
// deliver message to notify payment service to timeout a payment of the reservation
const (
	ReservationToPaymentDelayQueue        = "reservation.payment.timeout.delay"
	ReservationToPaymentTimeoutQueue      = "reservation.payment.timeout.immediate"
	ReservationToPaymentTimeoutExchange   = "reservation.timeout.exchange"
	ReservationToPaymentTimeoutRoutingKey = "reservation.timeout"
)

type ReservationToPaymentDelayMessage struct {
	ReservationID uint `json:"reservation_id"`
}

// immediate queue from payment to reservation db
// deliver message to notify reservation db to store a paid reservation
const (
	PaymentToOrderImmediateQueue = "payment.order.create.immediate"
)

type PaymentToOrderImmediateMessage struct {
	ReservationID uint `json:"reservation_id"`
}
