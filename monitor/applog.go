package monitor

import (
	"github.com/belogik/goes"
	"regexp"
)

const (
	APPLOG_INDEX_NAME = "applog-*"
)

type ApplogInfo struct {
	Timestamp string
	LogName   string
	Keyword   string
	Message   string
}

type MonitorApplog struct {
	esHost      string
	esPort      string
	esIndexName string
	esTypeName  string
}

func NewMonitorApplog(host, port, typeName string) *MonitorApplog {
	ma := new(MonitorApplog)
	ma.esHost = host
	ma.esPort = port
	ma.esIndexName = APPLOG_INDEX_NAME
	ma.esTypeName = typeName
	return ma
}

func (ma *MonitorApplog) GetApplogInfo(logName, keyword string, excludes string, from int64, size int) ([]*ApplogInfo, error) {
	query := map[string]interface{}{
		"sort": map[string]interface{}{
			"@timestamp": map[string]interface{}{
				"order": "asc",
			},
		},
		"from": 0,
		"size": 10,
		"query": map[string]interface{}{
			"query_string": map[string]interface{}{
				"query": "@log_name:" + logName + " AND message:*" + keyword + "*",
				"lowercase_expanded_terms": "false",
			},
		},
		"filter": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": map[string]interface{}{
					"range": map[string]interface{}{
						"@timestamp": map[string]interface{}{
							"gt": from,
						},
					},
				},
			},
		},
	}

	conn := goes.NewConnection(ma.esHost, ma.esPort)
	searchResponse, err := conn.Search(query, []string{ma.esIndexName}, []string{ma.esTypeName}, nil)
	if err != nil {
		return nil, err
	}

	hits := searchResponse.Hits.Hits
	if len(hits) == 0 {
		return make([]*ApplogInfo, 0), nil
	}

	array := make([]*ApplogInfo, 0)
	for _, hit := range hits {
		applogInfo := new(ApplogInfo)
		applogInfo.LogName = logName
		applogInfo.Keyword = keyword
		applogInfo.Message = hit.Source["message"].(string)
		applogInfo.Timestamp = hit.Source["@timestamp"].(string)

		if excludes != "" {
			r := regexp.MustCompile(excludes)
			if r.MatchString(applogInfo.Message) {
				continue;
			}
		}

		array = append(array, applogInfo)
	}

	return array, nil
}
