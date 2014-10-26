package main

import (
	influxdb "github.com/influxdb/influxdb/client"
	"github.com/thoj/go-ircevent"
	"log"
	"os"
)

var (
	server       = os.Getenv("IRC_SERVER")
	nick         = os.Getenv("IRC_NICK")
	channel      = os.Getenv("IRC_CHANNEL")
	influxdbHost = os.Getenv("INFLUXDB_HOST") + ":" + os.Getenv("INFLUXDB_PORT")
	influxdbName = os.Getenv("INFLUXDB_NAME")
	influxdbUser = os.Getenv("INFLUXDB_USER")
	influxdbPass = os.Getenv("INFLUXDB_PASS")
)

func main() {
	conf := &influxdb.ClientConfig{
		Host:     influxdbHost,
		Username: influxdbUser,
		Password: influxdbPass,
		Database: influxdbName,
	}
	client, err := influxdb.NewClient(conf)
	if err != nil {
		log.Fatal(err)
	}

	createDatabaseIfNotExists(client)

	conn := irc.IRC(nick, nick)
	if err := conn.Connect(server); err != nil {
		log.Fatal(err)
	}
	conn.AddCallback("001", func(e *irc.Event) { conn.Join(channel) })
	conn.AddCallback("PRIVMSG", func(event *irc.Event) {
		series := &influxdb.Series{
			Name:    "message",
			Columns: []string{"nick"},
			Points: [][]interface{}{
				{event.Nick},
			},
		}
		err := client.WriteSeries([]*influxdb.Series{series})
		if err != nil {
			log.Printf("Error writing series: %v", err)
		}
		log.Printf("EVT: Message from %s", event.Nick)
	})
	conn.AddCallback("JOIN", func(event *irc.Event) {
		if event.Nick == nick {
			log.Printf("Joined channel %s", channel)
			return
		}

		series := &influxdb.Series{
			Name:    "join",
			Columns: []string{"nick"},
			Points: [][]interface{}{
				{event.Nick},
			},
		}
		err := client.WriteSeries([]*influxdb.Series{series})
		if err != nil {
			log.Printf("Error writing series: %v", err)
		}
		log.Printf("EVT: Join from %s", event.Nick)
	})
	conn.Loop()
}

func createDatabaseIfNotExists(client *influxdb.Client) {
	if dbList, err := client.GetDatabaseList(); err == nil {
		dbExists := false
		for _, db := range dbList {
			if db["name"] == influxdbName {
				dbExists = true
				break
			}
		}

		if !dbExists {
			if err := client.CreateDatabase(influxdbName); err != nil {
				log.Fatalf("Could not create influxdb database: %v", err)
			} else {
				log.Printf("Created influxdb database: %s", influxdbName)
			}
		}
	} else {
		log.Fatalf("Could not access influxdb: %v", err)
	}
}
