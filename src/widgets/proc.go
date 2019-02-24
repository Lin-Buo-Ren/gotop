package widgets

import (
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gizak/termui"
	psCPU "github.com/shirou/gopsutil/cpu"

	ui "github.com/cjbassi/gotop/src/termui"
	"github.com/cjbassi/gotop/src/utils"
)

const (
	UP   = "▲"
	DOWN = "▼"
)

type Proc struct {
	Pid         int
	CommandName string
	FullCommand string
	Cpu         float64
	Mem         float64
}

type ProcWidget struct {
	*ui.Table
	cpuCount         float64
	updateInterval   time.Duration
	sortMethod       string
	groupedProcs     []Proc
	ungroupedProcs   []Proc
	showGroupedProcs bool
}

func NewProcWidget(renderLock *sync.RWMutex) *ProcWidget {
	cpuCount, err := psCPU.Counts(false)
	if err != nil {
		log.Printf("failed to get CPU count from gopsutil: %v", err)
	}
	self := &ProcWidget{
		Table:            ui.NewTable(),
		updateInterval:   time.Second,
		cpuCount:         float64(cpuCount),
		sortMethod:       "c",
		showGroupedProcs: true,
	}
	self.Title = " Processes "
	self.ColResizer = self.ColResize
	self.Cursor = true
	self.Gap = 3
	self.PadLeft = 2

	self.UniqueCol = 0
	if self.showGroupedProcs {
		self.UniqueCol = 1
	}

	self.update()

	go func() {
		for range time.NewTicker(self.updateInterval).C {
			renderLock.RLock()
			self.update()
			renderLock.RUnlock()
		}
	}()

	return self
}

// Sort sorts either the grouped or ungrouped []Process based on the sortMethod.
// Called with every update, when the sort method is changed, and when processes are grouped and ungrouped.
func (self *ProcWidget) Sort() {
	self.Header = []string{"Count", "Command", "CPU%", "Mem%"}

	if !self.showGroupedProcs {
		self.Header[0] = "PID"
	}

	processes := &self.ungroupedProcs
	if self.showGroupedProcs {
		processes = &self.groupedProcs
	}

	switch self.sortMethod {
	case "c":
		sort.Sort(sort.Reverse(SortProcsByCpu(*processes)))
		self.Header[2] += DOWN
	case "p":
		if self.showGroupedProcs {
			sort.Sort(sort.Reverse(SortProcsByPid(*processes)))
		} else {
			sort.Sort(SortProcsByPid(*processes))
		}
		self.Header[0] += DOWN
	case "m":
		sort.Sort(sort.Reverse(SortProcsByMem(*processes)))
		self.Header[3] += DOWN
	}

	self.Rows = FieldsToStrings(*processes, self.showGroupedProcs)
}

// ColResize overrides the default ColResize in the termui table.
func (self *ProcWidget) ColResize() {
	self.ColWidths = []int{
		5, utils.MaxInt(self.Inner.Dx()-26, 10), 4, 4,
	}
}

func (self *ProcWidget) ChangeSort(e termui.Event) {
	if self.sortMethod != e.ID {
		self.sortMethod = e.ID
		self.Top()
		self.Sort()
	}
}

func (self *ProcWidget) Tab() {
	self.showGroupedProcs = !self.showGroupedProcs
	if self.showGroupedProcs {
		self.UniqueCol = 1
	} else {
		self.UniqueCol = 0
	}
	self.Sort()
	self.Top()
}

// GroupProcs groupes a []Process based on command name.
// The first field changes from PID to count.
// CPU and Mem are added together for each Process.
func GroupProcs(procs []Proc) []Proc {
	groupedProcs := make(map[string]Proc)
	for _, proc := range procs {
		val, ok := groupedProcs[proc.CommandName]
		if ok {
			groupedProcs[proc.CommandName] = Proc{
				val.Pid + 1,
				val.CommandName,
				"",
				val.Cpu + proc.Cpu,
				val.Mem + proc.Mem,
			}
		} else {
			groupedProcs[proc.CommandName] = Proc{
				1,
				proc.CommandName,
				"",
				proc.Cpu,
				proc.Mem,
			}
		}
	}

	groupList := make([]Proc, len(groupedProcs))
	var i int
	for _, val := range groupedProcs {
		groupList[i] = val
		i++
	}

	return groupList
}

// FieldsToStrings converts a []Process to a [][]string
func FieldsToStrings(processes []Proc, grouped bool) [][]string {
	strings := make([][]string, len(processes))
	for i, process := range processes {
		strings[i] = make([]string, 4)
		strings[i][0] = strconv.Itoa(int(process.Pid))
		if grouped {
			strings[i][1] = process.CommandName
		} else {
			strings[i][1] = process.FullCommand
		}
		strings[i][2] = fmt.Sprintf("%4s", strconv.FormatFloat(process.Cpu, 'f', 1, 64))
		strings[i][3] = fmt.Sprintf("%4s", strconv.FormatFloat(float64(process.Mem), 'f', 1, 64))
	}
	return strings
}

// Kill kills a process or group of processes depending on if we're displaying the processes grouped or not.
func (self *ProcWidget) Kill() {
	self.SelectedItem = ""
	command := "kill"
	if self.UniqueCol == 1 {
		command = "pkill"
	}
	cmd := exec.Command(command, self.Rows[self.SelectedRow][self.UniqueCol])
	cmd.Start()
	cmd.Wait()
}

/////////////////////////////////////////////////////////////////////////////////
//                              []Process Sorting                              //
/////////////////////////////////////////////////////////////////////////////////

type SortProcsByCpu []Proc

// Len implements Sort interface
func (self SortProcsByCpu) Len() int {
	return len(self)
}

// Swap implements Sort interface
func (self SortProcsByCpu) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

// Less implements Sort interface
func (self SortProcsByCpu) Less(i, j int) bool {
	return self[i].Cpu < self[j].Cpu
}

type SortProcsByPid []Proc

// Len implements Sort interface
func (self SortProcsByPid) Len() int {
	return len(self)
}

// Swap implements Sort interface
func (self SortProcsByPid) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

// Less implements Sort interface
func (self SortProcsByPid) Less(i, j int) bool {
	return self[i].Pid < self[j].Pid
}

type SortProcsByMem []Proc

// Len implements Sort interface
func (self SortProcsByMem) Len() int {
	return len(self)
}

// Swap implements Sort interface
func (self SortProcsByMem) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

// Less implements Sort interface
func (self SortProcsByMem) Less(i, j int) bool {
	return self[i].Mem < self[j].Mem
}
