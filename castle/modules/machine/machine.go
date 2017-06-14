package machine

import (
	"encoding/json"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

func New() *Machine {
	m := &Machine{}
	m.timer = time.NewTimer(time.Duration(0))
	m.timer.Stop()
	m.settings.Interval = 5 * time.Second
	go m.check()
	return m
}

type Machine struct {
	updates  chan interface{}
	timer    *time.Timer
	settings struct {
		Interval time.Duration
	}
	lastCPUStat cpu.CPUTimesStat
	status      struct {
		CPU         float64 `json:"cpu"`
		DiskUsed    int64   `json:"diskUsed"`
		DiskTotal   int64   `json:"diskTotal"`
		MemoryUsed  int64   `json:"memoryUsed"`
		MemoryTotal int64   `json:"memoryTotal"`
		GoMemory    int64   `json:"goMemory"`
		GoRoutines  int     `json:"goRoutines"`
	}
}

func (a *Machine) ID() string {
	return "machine"
}

func (m *Machine) check() {
	first := true
	for {
		//wait here for <interval>
		//short-circuited by Set()
		m.timer.Reset(m.settings.Interval)
		<-m.timer.C
		//load
		m.loadStats(first)
		first = false
	}
}

func (m *Machine) loadStats(first bool) {
	//count cpu cycles between last count
	if first {
		if stats, err := cpu.CPUTimes(false); err == nil {
			m.lastCPUStat = stats[0]
			time.Sleep(2 * time.Second)
		}
	}
	if stats, err := cpu.CPUTimes(false); err == nil {
		stat := stats[0]
		total := totalCPUTime(stat)
		last := m.lastCPUStat
		lastTotal := totalCPUTime(last)
		if lastTotal != 0 {
			totalDelta := total - lastTotal
			if totalDelta > 0 {
				idleDelta := (stat.Iowait + stat.Idle) - (last.Iowait + last.Idle)
				usedDelta := (totalDelta - idleDelta)
				m.status.CPU = 100 * usedDelta / totalDelta
			}
		}
		m.lastCPUStat = stat
	}
	//count disk usage
	if stat, err := disk.DiskUsage("/"); err == nil {
		m.status.DiskUsed = int64(stat.Used)
		m.status.DiskTotal = int64(stat.Total)
	}
	//count memory usage
	if stat, err := mem.VirtualMemory(); err == nil {
		m.status.MemoryUsed = int64(stat.Used)
		m.status.MemoryTotal = int64(stat.Total)
	}
	//count total bytes allocated by the go runtime
	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)
	m.status.GoMemory = int64(memStats.Alloc)
	//count current number of goroutines
	m.status.GoRoutines = runtime.NumGoroutine()
	//done
	m.push()
}

func (m *Machine) Status(updates chan interface{}) {
	m.updates = updates
	m.push()
}

func (m *Machine) push() {
	if m.updates != nil {
		m.updates <- &m.status
	}
}

func (m *Machine) Get() interface{} {
	return &m.settings
}

func (m *Machine) Set(j json.RawMessage) error {
	if j != nil {
		if err := json.Unmarshal(j, &m.settings); err != nil {
			return err
		}
	}
	//do stuff
	m.timer.Reset(0)
	return nil
}

func totalCPUTime(t cpu.CPUTimesStat) float64 {
	total := t.User + t.System + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal + t.Guest + t.GuestNice + t.Idle
	return total
}
