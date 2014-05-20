package monitor

import (
	"fmt"
	"github.com/belogik/goes"
	"strconv"
	"strings"
)

const (
	ES_INDEX_NAME = "dstat-*"
	SCROLL_SIZE   = 1000
)

type DstatInfo struct {
	Timestamp string
	CpuUsr    int64
	CpuSystem int64
	DiskUsed  int64
	DiskFree  int64
	MemUsed   int64
	MemBuff   int64
	MemCach   int64
	MemFree   int64
}

type MonitorDstat struct {
	esHost      string
	esPort      string
	esIndexName string
	esTypeName  string
}

func NewMonitorDstat(host, port, typeName string) *MonitorDstat {
	md := new(MonitorDstat)
	md.esHost = host
	md.esPort = port
	md.esIndexName = ES_INDEX_NAME
	md.esTypeName = typeName
	return md
}

func (md *MonitorDstat) GetDstatInfo(size int) []*DstatInfo {
	query := map[string]interface{}{
		"sort": map[string]interface{}{
			"@timestamp": "desc",
		},
		"from": 0,
		"size": size * 20,
		"query": map[string]interface{}{
			"query_string": map[string]interface{}{
				"query": "*:*",
			},
		},
	}

	conn := goes.NewConnection(md.esHost, md.esPort)
	searchResponse, err := conn.Search(query, []string{md.esIndexName}, []string{md.esTypeName}, nil)
	if err != nil {
		return nil
	}

	hits := searchResponse.Hits.Hits
	return createInfo(hits, size)
}

func createInfo(hits []goes.Hit, size int) []*DstatInfo {
	infos := make([]*DstatInfo, size)
	count := 0
	var info *DstatInfo = nil

	for _, hit := range hits {
		if count >= size {
			break
		}

		timeStamp := hit.Source["@timestamp"].(string)
		stat := hit.Source["stat"].(string)

		tmpVal := hit.Source["value"].(string)
		tmpValIndex := strings.Index(tmpVal, ".")
		if tmpValIndex >= 0 {
			tmpVal = tmpVal[0:tmpValIndex]
		}
		value, err := strconv.ParseInt(tmpVal, 10, 64)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if info != nil && hit.Source["@timestamp"] == info.Timestamp {
			info.set(stat, value)
		} else {
			if info != nil {
				if info.isCompleteInfo() {
					infos[count] = info
					count++
				}
			}
			info = &DstatInfo{"0", -1, -1, -1, -1, -1, -1, -1, -1}
			info.set(stat, value)
			info.Timestamp = timeStamp
		}
	}

	if info != nil && info.isCompleteInfo() {
		infos[count] = info
		count++
	}

	if count == 0 {
		return nil //TODO error
	}

	return infos[:count]
}

func (info *DstatInfo) set(stat string, value int64) {
	switch stat {
	case "cpu-usr":
		info.CpuUsr = value
	case "cpu-sys":
		info.CpuSystem = value
	case "disk-used":
		info.DiskUsed = value
	case "disk-free":
		info.DiskFree = value
	case "mem-used":
		info.MemUsed = value
	case "mem-buff":
		info.MemBuff = value
	case "mem-cach":
		info.MemCach = value
	case "mem-free":
		info.MemFree = value
	}
}

func (info *DstatInfo) isCompleteInfo() bool {
	if info.CpuSystem == -1 {
		return false
	} else if info.CpuUsr == -1 {
		return false
	} else if info.DiskFree == -1 {
		return false
	} else if info.DiskUsed == -1 {
		return false
	} else if info.MemBuff == -1 {
		return false
	} else if info.MemCach == -1 {
		return false
	} else if info.MemFree == -1 {
		return false
	} else if info.MemUsed == -1 {
		return false
	}
	return true
}

// これはApacheのほうかな、、
func (md *MonitorDstat) Scroll() {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"query_string": map[string]interface{}{
				"query": "*:*",
			},
		},
	}

	conn := goes.NewConnection(md.esHost, md.esPort)
	scanResponse, err := conn.Scan(query, []string{md.esIndexName}, []string{md.esTypeName}, "1m", SCROLL_SIZE)
	if err != nil {
		return
	}

	scrollId := scanResponse.ScrollId
	for {
		scrollResponse, err := conn.Scroll(scrollId, "1m")
		if err != nil {
			fmt.Println(err)
		}
		scrollId = scrollResponse.ScrollId
		hits := scrollResponse.Hits.Hits
		for _, hit := range hits {
			fmt.Println(hit.Source["stat"])
		}
	}
}
