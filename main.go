package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ulyssessouza/clf-analyzer-server/data"
	"encoding/json"
	"github.com/ulyssessouza/clf-analyzer-server/http"
	"fmt"
)

const apiVersion= "/v1"

var addr = flag.String("addr", "localhost:8000", "http service address")
var interruptChan = make(chan os.Signal, 1)

func UpdateScoresLoop(conn *websocket.Conn, doneChannel *chan struct{}) {
	defer close(*doneChannel)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}

		var sectionScoreEntries []data.SectionScoreEntry
		json.Unmarshal(message, &sectionScoreEntries)

		var newScore []string
		for _, scoreEntry := range sectionScoreEntries {
			newScore = append(newScore, fmt.Sprintf("[%d] %s", scoreEntry.Hits, scoreEntry.Section))
		}
		scores = newScore
	}
}

func UpdateAlertsLoop(conn *websocket.Conn, doneChannel *chan struct{}) {
	defer close(*doneChannel)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}

		var alertEntries []http.AlertEntry
		json.Unmarshal(message, &alertEntries)

		var newAlert []string
		for _, alertEntry := range alertEntries {
			var listMsg string
			if alertEntry.Charge > alertEntry.Limit {
				listMsg = "[Overcharged](fg-red)"
			} else {
				listMsg = "[Back from overcharge](fg-green)"
			}
			newAlert = append(newAlert, fmt.Sprintf("[%s] %s", alertEntry.AlertTime.String(), listMsg))
		}

		if len(alertEntries) > 0 {
			firstAlertEntry := alertEntries[0]
			if firstAlertEntry.Charge > firstAlertEntry.Limit {
				alertStatus = fmt.Sprintf("[High traffic generated an alert - hits = %d, triggered at %s](fg-red) on a limit of %d",
					firstAlertEntry.Charge, firstAlertEntry.AlertTime, firstAlertEntry.Limit)
			} else {
				alertStatus = fmt.Sprintf("[Traffic is normal with %d messages in a limit of %d in the last 2 minutes](fg-green)",
					firstAlertEntry.Charge, firstAlertEntry.Limit)
			}
		}

		alerts = newAlert
	}
}

func getConnection(path string) *websocket.Conn{
	u := url.URL{Scheme: "ws", Host: *addr, Path: fmt.Sprintf("%s%s", apiVersion, path)}
	log.Printf("Connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	return conn
}

// Cleanly close the connection by sending a close message
func closeConn(conn *websocket.Conn) bool {
	log.Printf("Disconnecting...\n")
	err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
		return false
	}
	return true
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	signal.Notify(interruptChan, os.Interrupt)

	scoreConn := getConnection("/score")
	defer scoreConn.Close()
	scoreDoneChan := make(chan struct{})
	go UpdateScoresLoop(scoreConn, &scoreDoneChan)

	alertConn := getConnection("/alert")
	defer alertConn.Close()
	alertDoneChan := make(chan struct{})
	go UpdateAlertsLoop(alertConn, &alertDoneChan)

	go ShowUi()

	for {
		select {
		case <-scoreDoneChan:
			return
		case <-alertDoneChan:
			return
		case <-interruptChan:
			log.Println("Bye bye!")
			closeConn(scoreConn)
			closeConn(alertConn)
			select {
			case <-scoreDoneChan:
			case <-alertDoneChan:
			case <-time.After(time.Second):
			}
			select {
			case <-scoreDoneChan:
			case <-alertDoneChan:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
