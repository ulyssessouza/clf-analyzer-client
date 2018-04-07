package main

import (
	"flag"
	"log"
	"os"
	"fmt"
	"time"
	"os/signal"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/ulyssessouza/clf-analyzer-server/data"
	"github.com/ulyssessouza/clf-analyzer-server/http"
	"github.com/gizak/termui"
)

const apiVersion1 = "/v1"

var addr = flag.String("addr", "localhost:8000", "http service address")
var interruptChan = make(chan os.Signal, 1)

func UpdateScoresLoop(conn *websocket.Conn, doneChannel *chan struct{}) {
	defer conn.Close()
	defer close(*doneChannel)
	for {
		var sectionScoreEntries []data.SectionScoreEntry
		err := conn.ReadJSON(&sectionScoreEntries)
		if err != nil {
			return
		}
		conn.WriteJSON(http.Ack{Code: http.AckOK})

		var newScore []string
		for _, scoreEntry := range sectionScoreEntries {
			newScore = append(newScore, fmt.Sprintf("[%d] %s", scoreEntry.Hits, scoreEntry.Section))
		}
		scores = newScore
	}
}

func UpdateAlertsLoop(conn *websocket.Conn, doneChannel *chan struct{}) {
	defer conn.Close()
	defer close(*doneChannel)
	for {
		var alertEntries []http.AlertEntry
		err := conn.ReadJSON(&alertEntries)
		if err != nil {
			return
		}
		conn.WriteJSON(http.Ack{Code: http.AckOK})

		var newAlert []string
		for _, alertEntry := range alertEntries {
			var listMsg string
			if alertEntry.Charge > alertEntry.Limit {
				listMsg = "[Overcharged](fg-red)"
			} else {
				listMsg = "[Normal traffic](fg-green)"
			}

			formattedTime := alertEntry.AlertTime.Format("2006-01-02 15:04:05")

			newAlert = append(newAlert, fmt.Sprintf("[%s] %s", formattedTime, listMsg))
		}
		alerts = newAlert

		if len(alertEntries) > 0 {
			firstAlertEntry := alertEntries[0]
			if firstAlertEntry.Charge > firstAlertEntry.Limit {
				formattedTime := firstAlertEntry.AlertTime.Format("15:04:05")
				alertStatus = fmt.Sprintf("[High traffic generated an alert - hits = %d, triggered at %s](fg-red) on a limit of %d in the last 2 minutes",
					firstAlertEntry.Charge, formattedTime, firstAlertEntry.Limit)
			} else {
				alertStatus = fmt.Sprintf("[Traffic is normal with %d messages in a limit of %d in the last 2 minutes](fg-green)",
					firstAlertEntry.Charge, firstAlertEntry.Limit)
			}
		}
	}
}

func UpdateHitsLoop(conn *websocket.Conn, doneChannel *chan struct{}) {
	defer conn.Close()
	defer close(*doneChannel)
	for {
		var hitsEntries []float64
		err := conn.ReadJSON(&hitsEntries)
		if err != nil {
			break
		}

		conn.WriteJSON(http.Ack{Code: http.AckOK})
		hits = hitsEntries
	}
}

func getConn(path string) *websocket.Conn {
	u := url.URL{Scheme: "ws", Host: *addr, Path: fmt.Sprintf("%s%s", apiVersion1, path)}
	log.Printf("Connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	conn.WriteJSON(http.Ack{Code: http.AckOK})
	return conn
}

// Cleanly close the connection by sending a close message
func closeConn(conns... *websocket.Conn) bool {
	log.Printf("Disconnecting...\n")
	for _, conn := range conns {
		err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("write close:", err)
			return false
		}
	}
	return true
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	signal.Notify(interruptChan, os.Interrupt)

	scoreConn, alertConn, hitsConn := getConn("/score"), getConn("/alert"), getConn("/hits")
	defer scoreConn.Close()
	defer alertConn.Close()
	defer hitsConn.Close()

	hitsDoneChan, alertDoneChan, scoreDoneChan := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go UpdateScoresLoop(scoreConn, &scoreDoneChan)
	go UpdateAlertsLoop(alertConn, &alertDoneChan)
	go UpdateHitsLoop(hitsConn, &hitsDoneChan)

	go ShowUi()

	for {
		select {
		case <-scoreDoneChan:
			return
		case <-alertDoneChan:
			return
		case <-interruptChan:
			log.Println("Bye bye!")
			closeConn(scoreConn, alertConn)
			select {
			case <-scoreDoneChan:
			case <-alertDoneChan:
			case <-time.After(time.Second):
			}
			break
		}
	}

	termui.StopLoop()
	termui.Close()
}
