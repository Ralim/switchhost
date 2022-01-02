package termui

import "sort"

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
