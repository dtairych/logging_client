package loggingclient

import (
	"encoding/json"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type LogEntry struct {
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

type Logger struct {
	service string
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

func NewLogger(service string) *Logger {
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		log.Fatal("RABBITMQ_URL environment variable not set")
	}

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ:", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		log.Fatal("Failed to open a channel:", err)
	}

	queue, err := channel.QueueDeclare(
		"logs", // name
		true,   // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	if err != nil {
		log.Fatal("Failed to declare a queue:", err)
	}

	return &Logger{
		service: service,
		conn:    conn,
		channel: channel,
		queue:   queue,
	}
}

func (logger *Logger) Log(level, message string) {
	logEntry := LogEntry{
		Service:   logger.service,
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	logEntryBytes, err := json.Marshal(logEntry)
	if err != nil {
		log.Printf("Error marshalling log entry: %v", err)
		return
	}

	err = logger.channel.Publish(
		"",                // exchange
		logger.queue.Name, // routing key
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        logEntryBytes,
		})
	if err != nil {
		log.Printf("Error publishing log entry: %v", err)
	}
}

func (logger *Logger) Close() {
	logger.channel.Close()
	logger.conn.Close()
}
