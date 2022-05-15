package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/streadway/amqp"
)

var ctx = context.Background()
var rdb = RedisConnect()

func RedisConnect() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "172.17.0.2:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func receiveRPC() {
	conn, err := amqp.Dial("amqp://guest:guest@172.17.0.3:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"rpc_queue", // name
		false,       // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			// log.Printf("Received a message: %s", d.Body)
			body := string(d.Body)
			msgid := body[len("Message-ID: <") : strings.Index(body, "\n")-2]
			sdomain := getSenderDomain(body)
			time.Sleep(10 * time.Millisecond)
			log.Printf("Message-ID: %s \n", msgid)
			log.Printf("Sender Domain: %s", sdomain)

			response := isDomainSecure(sdomain)

			err = ch.Publish(
				"",        // exchange
				d.ReplyTo, // routing key
				false,     // mandatory
				false,     // immediate
				amqp.Publishing{
					ContentType:   "text/plain",
					CorrelationId: d.CorrelationId,
					Body:          []byte(strconv.FormatBool(response)),
				})

			d.Ack(false)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func isDomainSecure(domain string) bool {
	exists := false
	val, err := rdb.Get(ctx, domain).Result()
	if err != nil && err != redis.Nil {
		log.Printf("Redis error: %s", err.Error())
	}
	if err == redis.Nil {
		log.Printf("%s NOT in redis!", domain)
		exists = domainExists(domain)
		rdb.Set(ctx, domain, exists, 0)
	} else {
		log.Printf("%s IS in redis!", domain)
		exists, _ = strconv.ParseBool(val)
	}
	return exists
}

func domainExists(domain string) bool {
	dat, err := os.ReadFile("ALL-phishing-domains.txt")
	if err != nil && err != redis.Nil {
		log.Printf("Domain files error: %s", err.Error())
	}

	return strings.Contains(fmt.Sprint(dat), domain)
}
