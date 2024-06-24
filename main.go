package loggingclient

import (
	"encoding/json"
	"log"
	"os"
	"runtime"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type LogEntry struct {
	Type      string            `json:"type"`
	Service   string            `json:"service"`
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level,omitempty"`
	Message   string            `json:"message,omitempty"`
	File      string            `json:"file,omitempty"`
	Line      int               `json:"line,omitempty"`
	User      string            `json:"user,omitempty"`
	IP        string            `json:"ip,omitempty"`
	Resource  string            `json:"resource,omitempty"`
	Action    string            `json:"action,omitempty"`
	Status    string            `json:"status,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Payload   string            `json:"payload,omitempty"`
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
		log.Fatal("Failed to declare a logs queue:", err)
	}

	return &Logger{
		service: service,
		conn:    conn,
		channel: channel,
		queue:   queue,
	}
}

func (logger *Logger) LogSystem(level, message string) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	}

	logEntry := LogEntry{
		Type:      "system",
		Service:   logger.service,
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		File:      file,
		Line:      line,
	}

	logger.publish(logEntry)
}

func (logger *Logger) LogUser(user, ip, resource, action, status string) {
	logEntry := LogEntry{
		Type:      "user",
		Service:   logger.service,
		Timestamp: time.Now(),
		User:      user,
		IP:        ip,
		Resource:  resource,
		Action:    action,
		Status:    status,
	}

	logger.publish(logEntry)
}

func (logger *Logger) LogAPI(logType, ip, payload string, headers map[string]string) {
	logEntry := LogEntry{
		Type:      "api",
		Service:   logger.service,
		Timestamp: time.Now(),
		IP:        ip,
		Headers:   headers,
		Payload:   payload,
	}

	logger.publish(logEntry)
}

func (logger *Logger) publish(logEntry LogEntry) {
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
