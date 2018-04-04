package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/ulyssessouza/clf-analyzer-server/data"
	"encoding/json"
)

var addr = flag.String("addr", "localhost:8000", "http service address")
var interruptChan = make(chan os.Signal, 1)

func UpdateScoresLoop(conn *websocket.Conn, doneChannel *chan struct{}) {
	defer close(*doneChannel)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			continue
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
			continue
		}

		var alertEntries []data.AlertEntry
		json.Unmarshal(message, &alertEntries)

		var newAlert []string
		for _, alertEntry := range alertEntries {
			var msg string
			if alertEntry.Overcharged {
				msg = "[Overcharged](fg-red)"
			} else {
				msg = "[Back from overcharge](fg-green)"
			}
			newAlert = append(newAlert, fmt.Sprintf("[%s] %s", alertEntry.AlertTime.String(), msg))
		}
		alerts = newAlert
	}
}


func getConnection(path string) *websocket.Conn{
	u := url.URL{Scheme: "ws", Host: *addr, Path: path}
	log.Printf("Connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	return conn
}

// Cleanly close the connection by sending a close message and then
// waiting (with timeout) for the server to close the connection.
func closeConn(conn *websocket.Conn) bool {
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
	defer close(scoreDoneChan)
	go UpdateScoresLoop(scoreConn, &scoreDoneChan)

	alertConn := getConnection("/alert")
	defer alertConn.Close()
	alertDoneChan := make(chan struct{})
	defer close(alertDoneChan)
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
			if !closeConn(scoreConn) || !closeConn(alertConn){
				return
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
