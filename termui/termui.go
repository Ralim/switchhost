package termui

import (
	"sync"

	"github.com/rivo/tview"
)

// TermUI is the wrapper for the basic terminal interface provided
// It shows the logs redirected to the side, along with the program status

type TermUI struct {
	sync.Mutex
	running bool
	app     *tview.Application

	//Logger points to this
	LogsView   *tview.TextView
	Statistics *Statistics

	statusTable *tview.Table
	tasks       []*TaskState
}

func NewTermUI(noUI bool) *TermUI {

	t := &TermUI{
		tasks:   []*TaskState{},
		running: false,
	}
	if noUI {
		return t
	}
	t.app = tview.NewApplication()
	t.Statistics = newStatistics(t.app)

	//Logs stream

	t.LogsView = tview.NewTextView()
	t.LogsView.SetTextAlign(tview.AlignLeft)
	t.LogsView.SetDynamicColors(true)
	t.LogsView.SetChangedFunc(func() {
		t.app.Draw()
	})
	t.LogsView.SetMaxLines(4096)
	t.LogsView.SetWrap(true)
	t.LogsView.SetTitle("Log Stream")
	t.LogsView.SetBorder(true)

	//Status table

	t.statusTable = tview.NewTable()
	t.statusTable.SetBorders(true)
	t.statusTable.SetTitle("Worker Tasks")
	t.statusTable.SetFixed(1, 1)
	t.statusTable.SetCellSimple(0, 0, "Task")
	t.statusTable.SetCellSimple(0, 1, "Status")
	t.statusTable.GetCell(0, 1).SetExpansion(1) // Set second col to expand out

	// Grid

	grid := tview.NewGrid()
	grid.SetRows(-2, -8)
	grid.SetColumns(-4, -6)
	grid.SetBorders(true)

	// Grid contents
	grid.AddItem(t.Statistics.table, 0, 0, 1, 2, 0, 0, false)
	grid.AddItem(t.statusTable, 1, 0, 1, 1, 0, 0, false)
	grid.AddItem(t.LogsView, 1, 1, 1, 1, 0, 0, true)

	t.app.SetRoot(grid, true)
	t.app.SetFocus(grid)
	t.app.EnableMouse(true)
	return t
}

func (t *TermUI) Run() {
	t.running = true
	_ = t.app.Run()

}

func (t *TermUI) Stop() {
	t.running = false
	t.app.Stop()
}
