package main

import (
	"testing"

	"github.com/yfujita/monitoring-elasticsearch-fluent/monitor"
)

type rateTest struct {
	in                                  []*monitor.DstatInfo
	outCPURate, outDiskRate, outMemRate int64
}

var rateTests = []rateTest{
	rateTest{
		[]*monitor.DstatInfo{
			&monitor.DstatInfo{Timestamp: "0", CpuUsr: 1, CpuSystem: 1, DiskUsed: 1, DiskFree: 1, MemUsed: 1, MemBuff: 1, MemCach: 1, MemFree: 1},
		},
		2, 50, 25,
	},
}

func TestGetResourceUsageRate(t *testing.T) {
	for _, rt := range rateTests {
		cpuRate, diskRate, memRate := GetResourceUsageRate(rt.in)
		if cpuRate != rt.outCPURate || diskRate != rt.outDiskRate || memRate != rt.outMemRate {
			t.Errorf("Failed expect(%d %d %d) get(%d %d %d)",
				rt.outCPURate, rt.outDiskRate, rt.outMemRate, cpuRate, diskRate, memRate)
		}
	}
}
