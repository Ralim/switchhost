package termui

import "github.com/rivo/tview"

type TaskState struct {
	name        string
	lastStatus  string
	statusTable *tview.Table
	parent      *TermUI
	//Row and col of the status cell
	row int
	col int
}

func (t *TaskState) UpdateStatus(state string) {
	t.lastStatus = state
	if t.parent.running {
		t.parent.app.QueueUpdateDraw(func() {
			t.statusTable.SetCellSimple(t.row, t.col, state)
		})
	}
}

//redraw draws title and contents again
func (t *TaskState) redraw() {
	if t.parent.running {
		t.parent.app.QueueUpdateDraw(func() {
			t.statusTable.SetCellSimple(t.row, t.col, t.lastStatus)
			t.statusTable.SetCellSimple(t.row, t.col-1, t.name)
		})
	}
}
