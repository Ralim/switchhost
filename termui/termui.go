package termui

import (
	"github.com/rivo/tview"
)

type TermUI struct {
	app *tview.Application

	//Logger points to this
	LogsView *tview.TextView
}

func NewTermUI() *TermUI {

	box := tview.NewBox().SetBorder(true).SetTitle("Loading")
	t := &TermUI{}
	t.LogsView = tview.NewTextView()
	t.LogsView.SetText("Loading...\n")
	t.LogsView.SetTextAlign(tview.AlignLeft)
	t.LogsView.SetDynamicColors(true)
	t.LogsView.SetChangedFunc(func() {
		t.app.Draw()
	})
	t.LogsView.SetMaxLines(4096)
	t.LogsView.SetWrap(false)
	t.LogsView.SetTitle("Logs")
	t.LogsView.SetBorder(true)

	grid := tview.NewGrid()
	grid.SetRows(-1)
	grid.SetColumns(-1, -1)
	grid.SetBorders(true)

	// Layout for screens narrower than 100 cells (menu and side bar are hidden).
	grid.AddItem(box, 0, 0, 1, 1, 0, 0, true)
	grid.AddItem(t.LogsView, 0, 1, 1, 1, 0, 0, false)

	t.app = tview.NewApplication()
	t.app.SetRoot(grid, true)
	t.app.SetFocus(grid)
	return t
}

func (t *TermUI) Run() {
	t.app.Run()

}
func (t *TermUI) Stop() {
	t.app.Stop()

}
