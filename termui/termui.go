package termui

import (
	"sort"
	"sync"

	"github.com/rivo/tview"
)

// TermUI is the wrapper for the basic terminal interface provided
// It shows the logs redirected to the side, along with the program status

type TaskState struct {
	name        string
	lastStatus  string
	statusTable *tview.Table
	app         *tview.Application
	//Row and col of the status cell
	row int
	col int
}

type TermUI struct {
	sync.Mutex
	app *tview.Application

	//Logger points to this
	LogsView *tview.TextView

	statusTable *tview.Table
	tasks       []*TaskState
}

func NewTermUI() *TermUI {

	t := &TermUI{
		tasks: []*TaskState{},
	}

	//Logs stream

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

	//Status table

	t.statusTable = tview.NewTable()
	t.statusTable.SetBorders(true)
	t.statusTable.SetTitle("Status")
	t.statusTable.SetFixed(1, 1)
	t.statusTable.SetCellSimple(0, 0, "Task")
	t.statusTable.SetCellSimple(0, 1, "Status")

	// Grid

	grid := tview.NewGrid()
	grid.SetRows(-1)
	grid.SetColumns(-1, -1)
	grid.SetBorders(true)

	// Grid contents

	grid.AddItem(t.statusTable, 0, 0, 1, 1, 0, 0, true)
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

func (t *TermUI) sortTasks() {
	//Sorts tasks alphabetically and redraws the list
	sort.SliceStable(t.tasks, func(i, j int) bool {
		return t.tasks[i].name < t.tasks[j].name
	})
	for i := 0; i < len(t.tasks); i++ {
		t.tasks[i].row = i + 1
		t.tasks[i].redraw()
	}
}
func (t *TermUI) RegisterTask(taskName string) *TaskState {
	t.Lock()
	defer t.Unlock()
	index := len(t.tasks) + 1
	//Todo, support 2xN or 4xN ColsxRows
	row := index
	colindex := 0
	t.app.QueueUpdateDraw(func() {
		t.statusTable.SetCellSimple(row, colindex, taskName)
	})
	state := &TaskState{
		name:        taskName,
		statusTable: t.statusTable,
		app:         t.app,
		row:         row,
		col:         colindex + 1,
	}
	state.UpdateStatus("Loading...")
	t.tasks = append(t.tasks, state)
	t.sortTasks() // Ensure tasks are sorted

	return state
}

func (t *TaskState) UpdateStatus(state string) {
	t.lastStatus = state
	t.app.QueueUpdateDraw(func() {
		t.statusTable.SetCellSimple(t.row, t.col, state)
	})
}

//redraw draws title and contents again
func (t *TaskState) redraw() {
	t.app.QueueUpdateDraw(func() {
		t.statusTable.SetCellSimple(t.row, t.col, t.lastStatus)
		t.statusTable.SetCellSimple(t.row, t.col-1, t.name)
	})
}
