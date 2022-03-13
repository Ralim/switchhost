package termui

import (
	"fmt"

	"github.com/rivo/tview"
)

// Tracks library statistics
// Used to show an info panel at the bottom of the screen

type Statistics struct {
	TotalTitles  int
	TotalUpdates int
	TotalDLC     int

	table *tview.Table
	app   *tview.Application
}

func (s Statistics) Redraw() {
	newTitles := fmt.Sprintf("%d", s.TotalTitles)
	newUpdates := fmt.Sprintf("%d", s.TotalUpdates)
	newDLC := fmt.Sprintf("%d", s.TotalDLC)
	if s.app != nil {
		s.app.QueueUpdateDraw(func() {
			s.table.SetCellSimple(0, 0, "Total Titles")
			s.table.SetCellSimple(0, 1, newTitles)
			s.table.SetCellSimple(1, 0, "Total Updates")
			s.table.SetCellSimple(1, 1, newUpdates)
			s.table.SetCellSimple(2, 0, "Total DLC")
			s.table.SetCellSimple(2, 1, newDLC)
		})
	}
}

func newStatistics(app *tview.Application) *Statistics {
	s := &Statistics{app: app}

	s.table = tview.NewTable()
	s.table.SetBorders(true)
	s.table.SetTitle("Statistics")
	s.table.SetFixed(0, 1)
	s.table.SetCellSimple(0, 0, "Total Titles")
	s.table.SetCellSimple(0, 1, "0")
	s.table.SetCellSimple(1, 0, "Total Updates")
	s.table.SetCellSimple(1, 1, "0")
	s.table.SetCellSimple(2, 0, "Total DLC")
	s.table.SetCellSimple(2, 1, "0")
	return s
}
