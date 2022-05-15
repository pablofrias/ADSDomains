package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/payleet/zlogger"
	"github.com/streadway/amqp"
)

const pathRPC string = "./data/enron_mail_20150507/maildir/"
const inboxPathRPC string = "inbox"

var logRPC = *zlogger.NewLogger()

func readEmailRPC(file string) error {
	logRPC.Info(fmt.Sprintf("Reading %s", file))
	dat, err := os.ReadFile(file)
	if err != nil {
		logRPC.Error(err.Error())
		return err
	}
	sendEmailRPC(dat)
	return nil
}

func sendEmailRPC(email []byte) {
	logRPC.Info("Sending bytes to RabbitMQ")
	rabbitSendRPC(email)
}

func listFoldersRPC() {
	emailCounter := 0
	entries, err := os.ReadDir(pathRPC)
	if err != nil {
		logRPC.Error(err.Error())
		return
	}
	for _, entry := range entries {
		logRPC.Debug(fmt.Sprintf("Entry isdir: %t, name: %s \n", entry.IsDir(), entry.Name()))

		if entry.IsDir() {
			inbox, err := os.ReadDir(pathRPC + entry.Name() + "/inbox/")
			if err != nil {
				logRPC.Error(err.Error())
				continue
			}

			for _, email := range inbox {
				if !email.IsDir() {
					readEmailRPC(pathRPC + entry.Name() + "/inbox/" + email.Name())
					emailCounter++
					// time.Sleep(90 * time.Millisecond)
				}
			}
		}
	}
	logRPC.Info(fmt.Sprintf("Read %d emails", emailCounter))
}

func failOnErrorRPC(err error, msg string) {
	if err != nil {
		logRPC.Fatal(fmt.Sprintf("%s: %s", msg, err))
	}
}

func rabbitSendRPC(payload []byte) {
	conn, err := amqp.Dial("amqp://guest:guest@172.17.0.3:5672/")
	failOnErrorRPC(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnErrorRPC(err, "Failed to open a channel")
	defer ch.Close()

	responseQueue, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnErrorRPC(err, "Failed to declare a queue")

	q, err := ch.QueueDeclare(
		"rpc_queue", // name
		false,       // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	failOnErrorRPC(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		responseQueue.Name, // queue
		"",                 // consumer
		true,               // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	failOnErrorRPC(err, "Failed to register a consumer")

	corrId := randomString(32)

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			DeliveryMode:  amqp.Persistent,
			ContentType:   "text/plain",
			CorrelationId: corrId,
			ReplyTo:       responseQueue.Name,
			Body:          payload,
		})
	failOnErrorRPC(err, "Failed to publish a message")
	logRPC.Info(" [x] Sent payload - Waiting for response\n")

	for d := range msgs {
		if corrId == d.CorrelationId {
			res := string(d.Body)
			failOnErrorRPC(err, "Failed to convert body to integer")
			logRPC.Debug(fmt.Sprintf("Response: %s \n", res))
			break
		}
	}

	return
}

func randomString(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(65, 90))
	}
	return string(bytes)
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}
