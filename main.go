package main

import (
	"flag"
	"fmt"
	"github.com/yfujita/monitoring-elasticsearch-fluent/gmail"
	"github.com/yfujita/monitoring-elasticsearch-fluent/monitor"
	goyaml "gopkg.in/yaml.v1"
	"io/ioutil"
	"strconv"
)

const (
	CONF_ESHOST       = "eshost"
	CONF_ESPORT       = "esport"
	CONF_MAILTO       = "mailto"
	CONF_MAILFROM     = "mailfrom"
	CONF_PASSWORD     = "password"
	CONF_DSTAT        = "dstat"
	CONF_DSTAT_SERVER = "server"
)

type AlertInfo struct {
	title string
	msg   string
}

type Config struct {
	eshost       string
	esport       string
	mailTo       string
	mailFrom     string
	password     string
	dstatConfigs []DstatConfig
}

type DstatConfig struct {
	server string
}

func main() {
	fmt.Println("start")

	var path string
	flag.StringVar(&path, "conf", "blank", "config file path")
	flag.Parse()
	fmt.Println(path)
	config := loadConfig(path)

	dstatCh := make(chan *AlertInfo, 100)

	if len(config.dstatConfigs) > 0 {
		for _, dstatConfig := range config.dstatConfigs {
			go monitoringDstat(config.eshost, config.esport, dstatConfig.server, dstatCh)
		}
	}

	for {
		dstatAlert := <-dstatCh
		fmt.Println(dstatAlert)
		sendAlertMail(dstatAlert, config)
	}

	fmt.Println("finish")
}

func monitoringDstat(hostName, port, typeName string, ch chan *AlertInfo) {
	//TODO
	md := monitor.NewMonitorDstat(hostName, port, typeName)
	infos := md.GetDstatInfo(1)
	for _, info := range infos {
		ai := new(AlertInfo)
		ai.msg = strconv.FormatInt(info.DiskUsed, 10)
		ai.title = strconv.FormatInt(info.DiskFree, 10)
		ch <- ai
	}
}

func sendAlertMail(alertInfo *AlertInfo, config *Config) {
	gm := gmail.NewGmail(config.mailFrom, config.password)
	gm.Send(config.mailTo, alertInfo.title, alertInfo.msg)
}

func loadConfig(path string) *Config {
	yaml, err := ioutil.ReadFile(path)

	m := make(map[interface{}]interface{})
	err = goyaml.Unmarshal(yaml, &m)
	if err != nil {
		panic(err)
	}

	config := new(Config)
	config.eshost = m[CONF_ESHOST].(string)
	config.esport = m[CONF_ESPORT].(string)
	config.mailTo = m[CONF_MAILTO].(string)
	config.mailFrom = m[CONF_MAILFROM].(string)
	config.password = m[CONF_PASSWORD].(string)

	dstatConfigs := m[CONF_DSTAT].([]interface{})
	config.dstatConfigs = make([]DstatConfig, len(dstatConfigs))
	for i, dstatConfig := range dstatConfigs {
		config.dstatConfigs[i].server = dstatConfig.(map[interface{}]interface{})[CONF_DSTAT_SERVER].(string)
	}

	return config
}
