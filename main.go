package main

import (
	"flag"
	"fmt"
	"github.com/yfujita/monitoring-elasticsearch-fluent/gmail"
	"github.com/yfujita/monitoring-elasticsearch-fluent/monitor"
	goyaml "gopkg.in/yaml.v1"
	"io/ioutil"
	"strconv"
	"strings"
)

const (
	CONF_ESHOST                      = "eshost"
	CONF_ESPORT                      = "esport"
	CONF_ALERT                       = "alert"
	CONF_ALERT_MAILTO                = "mailto"
	CONF_ALERT_MAILFROM              = "mailfrom"
	CONF_ALERT_PASSWORD              = "password"
	CONF_ALERT_TMPL_DSTAT_CPU_TITLE  = "tmpl-dstat-cpu-title"
	CONF_ALERT_TMPL_DSTAT_CPU_MSG    = "tmpl-dstat-cpu-msg"
	CONF_ALERT_TMPL_DSTAT_DISK_TITLE = "tmpl-dstat-disk-title"
	CONF_ALERT_TMPL_DSTAT_DISK_MSG   = "tmpl-dstat-disk-msg"
	CONF_ALERT_TMPL_DSTAT_MEM_TITLE  = "tmpl-dstat-mem-title"
	CONF_ALERT_TMPL_DSTAT_MEM_MSG    = "tmpl-dstat-mem-msg"
	CONF_DSTAT                       = "dstat"
	CONF_DSTAT_SERVER                = "server"
	CONF_DSTAT_CPU                   = "cpu-threshold"
	CONF_DSTAT_DISK                  = "disk-threshold"
	CONF_DSTAT_MEM                   = "mem-threshold"
)

type AlertInfo struct {
	title string
	msg   string
}

type Config struct {
	eshost       string
	esport       string
	alertConfig  AlertConfig
	dstatConfigs []DstatConfig
}

type AlertConfig struct {
	mailTo             string
	mailFrom           string
	password           string
	tmplDstatCpuTitle  string
	tmplDstatCpuMsg    string
	tmplDstatDiskTitle string
	tmplDstatDiskMsg   string
	tmplDstatMemTitle  string
	tmplDstatMemMsg    string
}

type DstatConfig struct {
	server   string
	cpurate  int64
	diskrate int64
	memrate  int64
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
			go monitoringDstat(config.eshost, config.esport, config.alertConfig, dstatConfig, dstatCh)
		}
	}

	for {
		dstatAlert := <-dstatCh
		fmt.Println(dstatAlert)
		sendAlertMail(dstatAlert, config)
	}

	fmt.Println("finish")
}

func monitoringDstat(hostName, port string, alertConfig AlertConfig, dstatConfig DstatConfig, ch chan *AlertInfo) {
	typeName := dstatConfig.server
	num := 12

	md := monitor.NewMonitorDstat(hostName, port, typeName)
	infos := md.GetDstatInfo(num)

	var diskRateTotal int64 = 0
	var cpuRateTotal int64 = 0
	var memRateTotal int64 = 0

	for _, info := range infos {
		diskRateTotal += info.DiskUsed * 100 / (info.DiskUsed + info.DiskFree)
		cpuRateTotal += info.CpuUsr + info.CpuSystem
		memRateTotal += info.MemUsed * 100 / (info.MemUsed + info.MemBuff + info.MemCach + info.MemFree)
	}

	var diskRate int64 = diskRateTotal / (int64)(num)
	var cpuRate int64 = cpuRateTotal / (int64)(num)
	var memRate int64 = memRateTotal / (int64)(num)

	if (int64)(diskRate) >= dstatConfig.diskrate {
		ai := new(AlertInfo)
		ai.title = strings.Replace(
			strings.Replace(alertConfig.tmplDstatDiskTitle, "{server}", typeName, -1),
			"{value}", strconv.FormatInt(diskRate, 10), -1)
		ai.msg = strings.Replace(
			strings.Replace(alertConfig.tmplDstatDiskMsg, "{server}", typeName, -1),
			"{value}", strconv.FormatInt(diskRate, 10), -1)
		ch <- ai
	}
	if cpuRate >= dstatConfig.cpurate {
		ai := new(AlertInfo)
		ai.title = strings.Replace(
			strings.Replace(alertConfig.tmplDstatCpuTitle, "{server}", typeName, -1),
			"{value}", strconv.FormatInt(cpuRate, 10), -1)
		ai.msg = strings.Replace(
			strings.Replace(alertConfig.tmplDstatCpuMsg, "{server}", typeName, -1),
			"{value}", strconv.FormatInt(cpuRate, 10), -1)
		ch <- ai
	}
	if memRate >= dstatConfig.memrate {
		ai := new(AlertInfo)
		ai.title = strings.Replace(
			strings.Replace(alertConfig.tmplDstatMemTitle, "{server}", typeName, -1),
			"{value}", strconv.FormatInt(memRate, 10), -1)
		ai.msg = strings.Replace(
			strings.Replace(alertConfig.tmplDstatMemMsg, "{server}", typeName, -1),
			"{value}", strconv.FormatInt(memRate, 10), -1)
		ch <- ai
	}

}

func sendAlertMail(alertInfo *AlertInfo, config *Config) {
	gm := gmail.NewGmail(config.alertConfig.mailFrom, config.alertConfig.password)
	gm.Send(config.alertConfig.mailTo, alertInfo.title, alertInfo.msg)
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

	alertConfig := m[CONF_ALERT].(map[interface{}]interface{})
	config.alertConfig.mailTo = alertConfig[CONF_ALERT_MAILTO].(string)
	config.alertConfig.mailFrom = alertConfig[CONF_ALERT_MAILFROM].(string)
	config.alertConfig.password = alertConfig[CONF_ALERT_PASSWORD].(string)
	config.alertConfig.tmplDstatDiskTitle = alertConfig[CONF_ALERT_TMPL_DSTAT_DISK_TITLE].(string)
	config.alertConfig.tmplDstatDiskMsg = alertConfig[CONF_ALERT_TMPL_DSTAT_DISK_MSG].(string)
	config.alertConfig.tmplDstatCpuTitle = alertConfig[CONF_ALERT_TMPL_DSTAT_CPU_TITLE].(string)
	config.alertConfig.tmplDstatCpuMsg = alertConfig[CONF_ALERT_TMPL_DSTAT_CPU_MSG].(string)
	config.alertConfig.tmplDstatMemTitle = alertConfig[CONF_ALERT_TMPL_DSTAT_MEM_TITLE].(string)
	config.alertConfig.tmplDstatMemMsg = alertConfig[CONF_ALERT_TMPL_DSTAT_MEM_MSG].(string)

	dstatConfigs := m[CONF_DSTAT].([]interface{})
	config.dstatConfigs = make([]DstatConfig, len(dstatConfigs))
	for i, dstatConfig := range dstatConfigs {
		tmp := dstatConfig.(map[interface{}]interface{})
		config.dstatConfigs[i].server = tmp[CONF_DSTAT_SERVER].(string)
		config.dstatConfigs[i].cpurate = (int64)(tmp[CONF_DSTAT_CPU].(int))
		config.dstatConfigs[i].diskrate = (int64)(tmp[CONF_DSTAT_DISK].(int))
		config.dstatConfigs[i].memrate = (int64)(tmp[CONF_DSTAT_MEM].(int))
	}

	fmt.Println(config)
	return config
}
