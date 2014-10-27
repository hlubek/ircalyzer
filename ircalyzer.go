package main

import (
	influxdb "github.com/influxdb/influxdb/client"
	"github.com/thoj/go-ircevent"
	"log"
	"os"
	"strings"
)

var (
	server       = os.Getenv("IRC_SERVER")
	nick         = os.Getenv("IRC_NICK")
	channels     = strings.Split(os.Getenv("IRC_CHANNELS"), ",")
	influxdbHost = getInfluxdbHost()
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
	conn.AddCallback("001", func(e *irc.Event) {
		for _, channel := range channels {
			conn.Join(channel)
		}
	})
	conn.AddCallback("PRIVMSG", func(event *irc.Event) {
		var channel string
		if len(event.Arguments) == 0 {
			return
		} else {
			channel = event.Arguments[0]
		}

		series := &influxdb.Series{
			Name:    "message",
			Columns: []string{"nick", "channel"},
			Points: [][]interface{}{
				{event.Nick, channel},
			},
		}
		err := client.WriteSeries([]*influxdb.Series{series})
		if err != nil {
			log.Printf("Error writing series: %v", err)
		}
		log.Printf("EVT: Message from %s in channel %s", event.Nick, channel)
	})
	conn.AddCallback("JOIN", func(event *irc.Event) {
		var channel string
		if len(event.Arguments) == 0 {
			return
		} else {
			channel = event.Arguments[0]
		}

		if event.Nick == nick {
			log.Printf("Joined channel %s", channel)
			return
		}

		series := &influxdb.Series{
			Name:    "join",
			Columns: []string{"nick", "channel"},
			Points: [][]interface{}{
				{event.Nick, channel},
			},
		}
		err := client.WriteSeries([]*influxdb.Series{series})
		if err != nil {
			log.Printf("Error writing series: %v", err)
		}
		log.Printf("EVT: Join from %s in channel %s", event.Nick, channel)
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

func getInfluxdbHost() string {
	host := os.Getenv("INFLUXDB_HOST")
	if host != "" && host[0] == '$' {
		host = os.Getenv(host[1:])
	}
	port := os.Getenv("INFLUXDB_PORT")
	if port != "" && port[0] == '$' {
		port = os.Getenv(port[1:])
	}
	return host + ":" + port
}
