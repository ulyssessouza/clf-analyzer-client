package main

import (
	"os"
	"math"

	"github.com/gizak/termui"
	"time"
)

var scores []string
var alerts []string
var alertStatus string

func ShowUi() {
	var ticker = time.NewTicker(time.Second)

	err := termui.Init()
	if err != nil {
		panic(err)
	}
	defer termui.Close()

	sinps := (func() []float64 {
		n := 220
		ps := make([]float64, n)
		for i := range ps {
			ps[i] = 1 + math.Sin(float64(i)/5)
		}
		return ps
	})()

	alertStatusLine := termui.NewPar("")
	alertStatusLine.Text = "[Borderless Text](fg-red)"
	alertStatusLine.Height = 1
	alertStatusLine.Border = false

	instructionsLine := termui.NewPar("[Q]uit")
	instructionsLine.Height = 1
	instructionsLine.Border = false

	highScoresList := termui.NewList()
	highScoresList.ItemFgColor = termui.ColorWhite
	highScoresList.BorderLabel = "Most visited sections"
	highScoresList.Height = 12

	lastAlertsList := termui.NewList()
	lastAlertsList.ItemFgColor = termui.ColorWhite
	lastAlertsList.BorderLabel = "Last alerts"
	lastAlertsList.Height = 12

	lineChart := termui.NewLineChart()
	lineChart.BorderLabel = "dot-mode Line Chart"
	lineChart.Mode = "dot"
	lineChart.Data = sinps[4:]
	lineChart.Height = 12
	lineChart.AxesColor = termui.ColorWhite
	lineChart.LineColor = termui.ColorCyan | termui.AttrBold

	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(6, 0, highScoresList),
			termui.NewCol(6, 0, lastAlertsList)),
		termui.NewRow(
			termui.NewCol(12, 0, lineChart),
		),
		termui.NewRow(
			termui.NewCol(10, 0, alertStatusLine),
			termui.NewCol(2, 0, instructionsLine),

		))

	go func () {
		for {
			<-ticker.C

			highScoresList.Items = scores
			lastAlertsList.Items = alerts
			alertStatusLine.Text = alertStatus

			termui.Body.Align()
			termui.Render(termui.Body)
		}
	}()
	termui.Handle("/sys/kbd/q", func(termui.Event) {
		termui.StopLoop()
		interruptChan <- os.Interrupt
	})
	termui.Loop()
}

