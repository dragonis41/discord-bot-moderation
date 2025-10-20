package utils

import (
	"fmt"
	"os"
	"runtime"

	"github.com/shirou/gopsutil/v3/process"
)

type SystemStats struct {
	process *process.Process
}

var systemStats *SystemStats

func InitSystemStats() error {
	pid := int32(os.Getpid())
	proc, err := process.NewProcess(pid)
	if err != nil {
		return err
	}
	systemStats = &SystemStats{
		process: proc,
	}
	return nil
}

func GetMemoryUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Convert bytes to MB
	allocMB := float64(m.Alloc) / 1024 / 1024
	totalAllocMB := float64(m.TotalAlloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024

	return fmt.Sprintf("Alloc: %.2f MB | TotalAlloc: %.2f MB | Sys: %.2f MB | GC: %d",
		allocMB, totalAllocMB, sysMB, m.NumGC)
}

func GetCPUUsage() string {
	if systemStats == nil || systemStats.process == nil {
		return "CPU stats not initialized"
	}

	// Get CPU percent since last call
	percent, err := systemStats.process.CPUPercent()
	if err != nil {
		return fmt.Sprintf("Error getting CPU usage: %v", err)
	}

	return fmt.Sprintf("%.2f%%", percent)
}

func GetDetailedStats() map[string]string {
	stats := make(map[string]string)

	// Memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats["Memory Allocated"] = fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024)
	stats["Total Memory Allocated"] = fmt.Sprintf("%.2f MB", float64(m.TotalAlloc)/1024/1024)
	stats["System Memory"] = fmt.Sprintf("%.2f MB", float64(m.Sys)/1024/1024)
	stats["GC Runs"] = fmt.Sprintf("%d", m.NumGC)
	stats["Goroutines"] = fmt.Sprintf("%d", runtime.NumGoroutine())

	// CPU stats
	if systemStats != nil && systemStats.process != nil {
		if percent, err := systemStats.process.CPUPercent(); err == nil {
			stats["CPU Usage"] = fmt.Sprintf("%.2f%%", percent)
		}

		if memInfo, err := systemStats.process.MemoryInfo(); err == nil {
			stats["RSS Memory"] = fmt.Sprintf("%.2f MB", float64(memInfo.RSS)/1024/1024)
			stats["VMS Memory"] = fmt.Sprintf("%.2f MB", float64(memInfo.VMS)/1024/1024)
		}
	}

	return stats
}
