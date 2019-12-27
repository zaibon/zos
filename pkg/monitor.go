package pkg

//go:generate mkdir -p stubs
//go:generate zbusc -module monitor -version 0.0.1 -name system -package stubs github.com/threefoldtech/zos/pkg+SystemMonitor stubs/system_monitor_stub.go
//go:generate zbusc -module monitor -version 0.0.1 -name host -package stubs github.com/threefoldtech/zos/pkg+HostMonitor stubs/host_monitor_stub.go

import (
	"context"
	"fmt"
	"time"

	"github.com/blang/semver"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/vishvananda/netlink"
)

// VirtualMemoryStat struct
type VirtualMemoryStat mem.VirtualMemoryStat

// DiskIOCountersStat struct
type DiskIOCountersStat struct {
	disk.IOCountersStat
	Time time.Time
}

func (s *DiskIOCountersStat) String() string {
	return fmt.Sprintf(
		"Read: (%d Bytes, %d Op), Write: (%d Bytes, %d Op)",
		s.ReadBytes, s.ReadCount,
		s.WriteBytes, s.WriteCount,
	)
}

//TimesStat struct
type TimesStat struct {
	cpu.TimesStat
	Percent float64
	Time    time.Time
}

func (s *TimesStat) String() string {
	return fmt.Sprintf("CPU: %s Percent: %0.00f (U: %0.00f, S: %0.00f, I: %0.00f)",
		s.CPU, s.Percent, s.User, s.System, s.Idle)
}

// CPUTimesStat alias for []TimesStat required by zbus
type CPUTimesStat []TimesStat

// DisksIOCountersStat alias for map[string]IOCountersStat required by zbus
type DisksIOCountersStat map[string]DiskIOCountersStat

// NicIOCounterStat counter for a nic
type NicIOCounterStat net.IOCountersStat

// NicsIOCounterStat alias for []NicIOCounterStat
type NicsIOCounterStat []NicIOCounterStat

// NetlinkAddress alias
type NetlinkAddress netlink.Addr

// NetlinkAddresses alias for [][]NetlinkAddress
type NetlinkAddresses []NetlinkAddress

//SystemMonitor interface
type SystemMonitor interface {
	Memory(ctx context.Context) <-chan VirtualMemoryStat
	CPU(ctx context.Context) <-chan CPUTimesStat
	Disks(ctx context.Context) <-chan DisksIOCountersStat
	Nics(ctx context.Context) <-chan NicsIOCounterStat
}

// HostMonitor interface
type HostMonitor interface {
	Version(ctx context.Context) <-chan semver.Version
	IPs(ctx context.Context) <-chan NetlinkAddresses
}
