package collectors

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type Metrics struct {
	CPU float64

	MemTotal       float64
	MemUsed        float64
	MemUsedPercent float64

	SwapTotal float64
	SwapUsed  float64

	DiskTotal       float64
	DiskUsed        float64
	DiskUsedPercent float64

	RxBytes uint64
	TxBytes uint64

	Uptime uint64

	Load1 float64

	LoggedInUsers    int
	TotalProcesses   uint64
	RunningProcesses int
}

func Collect(ctx context.Context) (*Metrics, error) {
	cpuPerc, _ := cpu.PercentWithContext(ctx, 0, false)

	vm, _ := mem.VirtualMemoryWithContext(ctx)
	sm, _ := mem.SwapMemoryWithContext(ctx)
	du, _ := disk.UsageWithContext(ctx, "/")
	netIO, _ := net.IOCountersWithContext(ctx, false)
	hi, _ := host.InfoWithContext(ctx)
	la, _ := load.AvgWithContext(ctx)
	users, _ := host.UsersWithContext(ctx)

	// Best effort for running processes (procs running / total)
	// gopsutil host.Info returns Procs (total processes)
	// distinguishing "Running" might need process iteration or load info.
	// For now we will use 0 for RunningProcesses if easier, or check load.
	// actually host.Info doesn't give "running".
	// We'll stick to TotalProcesses from host.Info for now.

	m := &Metrics{
		CPU:             cpuPerc[0],
		MemTotal:        float64(vm.Total) / 1024 / 1024,
		MemUsed:         float64(vm.Used) / 1024 / 1024,
		MemUsedPercent:  vm.UsedPercent,
		SwapTotal:       float64(sm.Total) / 1024 / 1024,
		SwapUsed:        float64(sm.Used) / 1024 / 1024,
		DiskTotal:       float64(du.Total) / 1024 / 1024 / 1024,
		DiskUsed:        float64(du.Used) / 1024 / 1024 / 1024,
		DiskUsedPercent: du.UsedPercent,
		RxBytes:         netIO[0].BytesRecv,
		TxBytes:         netIO[0].BytesSent,
		Uptime:          hi.Uptime,
		Load1:           la.Load1,
		LoggedInUsers:   len(users),
		TotalProcesses:  hi.Procs,
	}

	return m, nil
}

func Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
