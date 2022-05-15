package main

import (
	"fmt"
	"os"

	"github.com/payleet/zlogger"
	"github.com/streadway/amqp"
)

const path string = "./data/enron_mail_20150507/maildir/"
const inboxPath string = "inbox"

var log = *zlogger.NewLogger()

func readEmail(file string) error {
	log.Info(fmt.Sprintf("Reading %s", file))
	dat, err := os.ReadFile(file)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	sendEmail(dat)
	return nil
}

func sendEmail(email []byte) {
	log.Info("Sending bytes to RabbitMQ")
	rabbitSend(email)
}

func listFolders() {
	emailCounter := 0
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Error(err.Error())
		return
	}
	for _, entry := range entries {
		log.Debug(fmt.Sprintf("Entry isdir: %t, name: %s \n", entry.IsDir(), entry.Name()))

		if entry.IsDir() {
			inbox, err := os.ReadDir(path + entry.Name() + "/inbox/")
			if err != nil {
				log.Error(err.Error())
				continue
			}

			for _, email := range inbox {
				if !email.IsDir() {
					readEmail(path + entry.Name() + "/inbox/" + email.Name())
					emailCounter++
					// time.Sleep(90 * time.Millisecond)
				}
			}
		}
	}
	log.Info(fmt.Sprintf("Read %d emails", emailCounter))
}

func main() {
	// listFolders()
	listFoldersRPC()
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatal(fmt.Sprintf("%s: %s", msg, err))
	}
}

func rabbitSend(payload []byte) {
	conn, err := amqp.Dial("amqp://guest:guest@172.17.0.2:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"emails", // name
		false,    // durable
		false,    // delete when unused
		false,    // exclusive
		false,    // no-wait
		nil,      // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         payload,
		})
	failOnError(err, "Failed to publish a message")
	log.Info(" [x] Sent payload\n")
}
