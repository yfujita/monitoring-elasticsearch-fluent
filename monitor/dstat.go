package monitor

import (
	l4g "github.com/alecthomas/log4go"
	"errors"
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
	l4g.Info("New MonitorDstat. host:%s port:%s index:%s type:%s", md.esHost, md.esPort, md.esIndexName, md.esTypeName)
	return md
}

func (md *MonitorDstat) GetDstatInfo(from int64, size int) ([]*DstatInfo, error) {
	query := map[string]interface{}{
		"sort": map[string]interface{}{
			"@timestamp": map[string]interface{}{
				"order": "desc",
				//"ignore_unmapped": true,
			},
		},
		"from": 0,
		"size": size * 20,
		"query": map[string]interface{}{
			"query_string": map[string]interface{}{
				"query": "*:*",
			},
		},
		"filter": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": map[string]interface{}{
					"range": map[string]interface{}{
						"@timestamp": map[string]interface{}{
							"from": from,
						},
					},
				},
			},
		},
	}

	conn := goes.NewConnection(md.esHost, md.esPort)
	searchResponse, err := conn.Search(query, []string{md.esIndexName}, []string{md.esTypeName}, nil)
	if err != nil {
		return nil, err
	}

	hits := searchResponse.Hits.Hits
	if len(hits) == 0 {
		return nil, errors.New(fmt.Sprintf(
			"Dstat logs are not found. index:%s, type:%s", md.esIndexName, md.esTypeName))
	}

	return createInfo(hits, size), nil
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
			l4g.Error(err)
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
