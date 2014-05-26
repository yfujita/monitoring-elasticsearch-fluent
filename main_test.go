package main

import (
	"github.com/yfujita/monitoring-elasticsearch-fluent/monitor"
	"testing"
)

type rateTest struct {
	in                                  []*monitor.DstatInfo
	outCpuRate, outDiskRate, outMemRate int64
}

var rateTests = []rateTest{
	rateTest{
		[]*monitor.DstatInfo{
			&monitor.DstatInfo{"0", 1, 1, 1, 1, 1, 1, 1, 1},
		},
		2, 50, 25,
	},
}

func TestGetResourceUsageRate(t *testing.T) {
	for _, rt := range rateTests {
		cpuRate, diskRate, memRate := GetResourceUsageRate(rt.in)
		if cpuRate != rt.outCpuRate || diskRate != rt.outDiskRate || memRate != rt.outMemRate {
			t.Errorf("Failed expect(%d %d %d) get(%d %d %d)",
				rt.outCpuRate, rt.outDiskRate, rt.outMemRate, cpuRate, diskRate, memRate)
		}
	}
}
