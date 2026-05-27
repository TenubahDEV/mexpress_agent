package collectors

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
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

func getLoggedInUsersCount(ctx context.Context) int {
	// Intento principal usando gopsutil
	users, err := host.UsersWithContext(ctx)
	if err == nil && len(users) > 0 {
		return len(users)
	}

	// Fallback para Linux/Unix/macOS usando el comando "who"
	if runtime.GOOS != "windows" {
		cmd := exec.CommandContext(ctx, "who")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			count := 0
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					count++
				}
			}
			return count
		}
	} else {
		// Fallback para Windows usando "query user"
		cmd := exec.CommandContext(ctx, "query", "user")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			count := 0
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					count++
				}
			}
			if count > 1 {
				return count - 1 // Restamos 1 por la cabecera
			}
		}
	}

	return 0
}

func Collect(ctx context.Context) (*Metrics, error) {
	cpuPerc, _ := cpu.PercentWithContext(ctx, 0, false)

	vm, _ := mem.VirtualMemoryWithContext(ctx)
	sm, _ := mem.SwapMemoryWithContext(ctx)
	du, _ := disk.UsageWithContext(ctx, "/")
	netIO, _ := net.IOCountersWithContext(ctx, false)
	hi, _ := host.InfoWithContext(ctx)
	la, _ := load.AvgWithContext(ctx)

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
		LoggedInUsers:   getLoggedInUsersCount(ctx),
		TotalProcesses:  hi.Procs,
	}

	return m, nil
}

func Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
