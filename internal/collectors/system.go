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

	MemTotal float64
	MemUsed  float64

	DiskUsedPercent float64

	RxBytes uint64
	TxBytes uint64

	Uptime uint64

	Load1 float64
}

func Collect(ctx context.Context) (*Metrics, error) {
	cpuPerc, _ := cpu.PercentWithContext(ctx, 0, false)

	vm, _ := mem.VirtualMemoryWithContext(ctx)
	du, _ := disk.UsageWithContext(ctx, "/")
	netIO, _ := net.IOCountersWithContext(ctx, false)
	hi, _ := host.InfoWithContext(ctx)
	la, _ := load.AvgWithContext(ctx)

	m := &Metrics{
		CPU:             cpuPerc[0],
		MemTotal:        float64(vm.Total) / 1024 / 1024,
		MemUsed:         float64(vm.Used) / 1024 / 1024,
		DiskUsedPercent: du.UsedPercent,
		RxBytes:         netIO[0].BytesRecv,
		TxBytes:         netIO[0].BytesSent,
		Uptime:          hi.Uptime,
		Load1:           la.Load1,
	}

	return m, nil
}

func Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
