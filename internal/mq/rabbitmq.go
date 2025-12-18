package mq

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func InitQueues(mqConn *amqp.Connection) error {
	ch, err := NewChannel(mqConn)
	if err != nil {
		return err
	}
	defer ch.Close()

	// setup all needed queues(list in constants)
	if err := SetupImmediateQueue(ch, ReservationToPaymentImmediateQueue); err != nil {
		return err
	}
	if err := SetupDelayQueue(ch, ReservationToPaymentDelayQueue, ReservationToPaymentTimeoutExchange,
		ReservationToPaymentTimeoutQueue, ReservationToPaymentTimeoutRoutingKey); err != nil {
		return err
	}
	if err := SetupImmediateQueue(ch, PaymentToOrderImmediateQueue); err != nil {
		return err
	}

	// clear all leftover messages in the queues from the previous runs
	ClearQueue(mqConn, ReservationToPaymentImmediateQueue)
	ClearQueue(mqConn, ReservationToPaymentDelayQueue)
	ClearQueue(mqConn, ReservationToPaymentTimeoutQueue)
	ClearQueue(mqConn, PaymentToOrderImmediateQueue)

	return nil
}

func NewMQConn(url string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func NewChannel(conn *amqp.Connection) (*amqp.Channel, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	return ch, nil
}

func SetupImmediateQueue(ch *amqp.Channel, immediateQueueName string) error {
	_, err := ch.QueueDeclare(immediateQueueName, true, false, false, false, nil)
	return err
}

// the delay queue consists three part: delay queue, timeout exchange, timeout queue
// produce to the delay queue, and consume from the timeout queue
func SetupDelayQueue(ch *amqp.Channel, delayQueueName, timeoutExchangeName, timeoutQueueName string, timeoutRoutingKey string) error {
	delayArgs := amqp.Table{
		"x-message-ttl":             int32(15 * 60 * 1000), // 15 mins
		"x-dead-letter-exchange":    timeoutExchangeName,
		"x-dead-letter-routing-key": timeoutRoutingKey,
	}

	if _, err := ch.QueueDeclare(
		delayQueueName, true, false, false, false, delayArgs); err != nil {
		return err
	}

	if err := ch.ExchangeDeclare(timeoutExchangeName, "direct", true, false, false, false, nil); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(timeoutQueueName, true, false, false, false, nil); err != nil {
		return err
	}

	return ch.QueueBind(timeoutQueueName, timeoutRoutingKey, timeoutExchangeName, false, nil)
}

func ClearQueue(conn *amqp.Connection, queueName string) error {
	ch, err := NewChannel(conn)
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueuePurge(queueName, false)

	return err
}
