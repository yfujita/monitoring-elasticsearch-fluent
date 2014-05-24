package main

import (
	l4g "code.google.com/p/log4go"
	"flag"
	"github.com/yfujita/monitoring-elasticsearch-fluent/mail"
	"github.com/yfujita/monitoring-elasticsearch-fluent/monitor"
	goyaml "gopkg.in/yaml.v1"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	CONF_FILE                               = "mef.yml"
	LOG4G_XML                               = "log4go.xml"
	CONF_ESHOST                             = "eshost"
	CONF_ESPORT                             = "esport"
	CONF_ALERT                              = "alert"
	CONF_ALERT_SMTP_SERVER                  = "smtpserver"
	CONF_ALERT_SMTP_PORT                    = "smtpport"
	CONF_ALERT_MAILTO                       = "mailto"
	CONF_ALERT_MAILFROM                     = "mailfrom"
	CONF_ALERT_PASSWORD                     = "password"
	CONF_ALERT_TMPL_DSTAT_CPU_TITLE         = "tmpl-dstat-cpu-title"
	CONF_ALERT_TMPL_DSTAT_CPU_MSG           = "tmpl-dstat-cpu-msg"
	CONF_ALERT_TMPL_DSTAT_CPU_NORMAL_TITLE  = "tmpl-dstat-cpu-normal-title"
	CONF_ALERT_TMPL_DSTAT_CPU_NORMAL_MSG    = "tmpl-dstat-cpu-normal-msg"
	CONF_ALERT_TMPL_DSTAT_DISK_TITLE        = "tmpl-dstat-disk-title"
	CONF_ALERT_TMPL_DSTAT_DISK_MSG          = "tmpl-dstat-disk-msg"
	CONF_ALERT_TMPL_DSTAT_DISK_NORMAL_TITLE = "tmpl-dstat-disk-normal-title"
	CONF_ALERT_TMPL_DSTAT_DISK_NORMAL_MSG   = "tmpl-dstat-disk-normal-msg"
	CONF_ALERT_TMPL_DSTAT_MEM_TITLE         = "tmpl-dstat-mem-title"
	CONF_ALERT_TMPL_DSTAT_MEM_MSG           = "tmpl-dstat-mem-msg"
	CONF_ALERT_TMPL_DSTAT_MEM_NORMAL_TITLE  = "tmpl-dstat-mem-normal-title"
	CONF_ALERT_TMPL_DSTAT_MEM_NORMAL_MSG    = "tmpl-dstat-mem-normal-msg"
	CONF_DSTAT                              = "dstat"
	CONF_DSTAT_SERVER                       = "server"
	CONF_DSTAT_CPU                          = "cpu-threshold"
	CONF_DSTAT_DISK                         = "disk-threshold"
	CONF_DSTAT_MEM                          = "mem-threshold"
	CONF_DSTAT_INTERVAL                     = "interval"
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
	smtpServer               string
	smtpPort                 string
	mailTo                   string
	mailFrom                 string
	password                 string
	tmplDstatCpuTitle        string
	tmplDstatCpuMsg          string
	tmplDstatCpuNormalTitle  string
	tmplDstatCpuNormalMsg    string
	tmplDstatDiskTitle       string
	tmplDstatDiskMsg         string
	tmplDstatDiskNormalTitle string
	tmplDstatDiskNormalMsg   string
	tmplDstatMemTitle        string
	tmplDstatMemMsg          string
	tmplDstatMemNormalTitle  string
	tmplDstatMemNormalMsg    string
}

type DstatConfig struct {
	server   string
	cpurate  int64
	diskrate int64
	memrate  int64
	interval int64
}

func main() {
	var confPath string
	flag.StringVar(&confPath, "conf", "blank", "config file path")
	flag.Parse()
	confPath = path.Clean(confPath)

	config := loadConfig(confPath + "/" + CONF_FILE)
	l4g.LoadConfiguration(confPath + "/" + LOG4G_XML)
	defer l4g.Close()

	l4g.Info("start monitoring")

	dstatCh := make(chan *AlertInfo, 100)

	if len(config.dstatConfigs) > 0 {
		for _, dstatConfig := range config.dstatConfigs {
			go monitoringDstat(config.eshost, config.esport, config.alertConfig, dstatConfig, dstatCh)
		}
	}

	go func(ch chan *AlertInfo, config *Config) {
		for {
			ai := <-ch
			sendAlertMail(ai, config)
		}
	}(dstatCh, config)

	for {
		//TODO
		time.Sleep(1 * time.Hour)
	}
	l4g.Info("finish")
}

func monitoringDstat(hostName, port string, alertConfig AlertConfig, dstatConfig DstatConfig, ch chan *AlertInfo) {
	typeName := dstatConfig.server
	num := 12

	md := monitor.NewMonitorDstat(hostName, port, typeName)
	diskAlert := false
	cpuAlert := false
	memAlert := false

	for {
		l4g.Debug("start monitoring task. server:%s", typeName)

		infos, err := md.GetDstatInfo(num)
		if err != nil {
			l4g.Error(err.Error())
			time.Sleep((time.Duration)(dstatConfig.interval) * time.Second)
			continue
		}

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

		if (int64)(diskRate) >= dstatConfig.diskrate && !diskAlert {
			diskAlert = true
			ch <- NewAlertInfo(alertConfig.tmplDstatDiskTitle, alertConfig.tmplDstatDiskMsg,
				typeName, strconv.FormatInt(diskRate, 10))
		} else if (int64)(diskRate) < dstatConfig.diskrate && diskAlert {
			diskAlert = false
			ch <- NewAlertInfo(alertConfig.tmplDstatDiskNormalTitle, alertConfig.tmplDstatDiskNormalMsg,
				typeName, strconv.FormatInt(diskRate, 10))
		}

		if cpuRate >= dstatConfig.cpurate && !cpuAlert {
			cpuAlert = true
			ch <- NewAlertInfo(alertConfig.tmplDstatCpuTitle, alertConfig.tmplDstatCpuMsg,
				typeName, strconv.FormatInt(cpuRate, 10))
		} else if cpuRate < dstatConfig.cpurate && cpuAlert {
			cpuAlert = false
			ch <- NewAlertInfo(alertConfig.tmplDstatCpuNormalTitle, alertConfig.tmplDstatCpuNormalMsg,
				typeName, strconv.FormatInt(cpuRate, 10))
		}

		if memRate >= dstatConfig.memrate && !memAlert {
			memAlert = true
			ch <- NewAlertInfo(alertConfig.tmplDstatMemTitle, alertConfig.tmplDstatMemMsg,
				typeName, strconv.FormatInt(memRate, 10))
		} else if memRate < dstatConfig.memrate && memAlert {
			memAlert = false
			ch <- NewAlertInfo(alertConfig.tmplDstatMemNormalTitle, alertConfig.tmplDstatMemNormalMsg,
				typeName, strconv.FormatInt(memRate, 10))
		}
		time.Sleep((time.Duration)(dstatConfig.interval) * time.Second)
	}

}

func NewAlertInfo(titleTemplate, msgTemplate, server, value string) *AlertInfo {
	ai := new(AlertInfo)
	ai.title = strings.Replace(
		strings.Replace(titleTemplate, "{server}", server, -1),
		"{value}", value, -1)
	ai.msg = strings.Replace(
		strings.Replace(msgTemplate, "{server}", server, -1),
		"{value}", value, -1)
	return ai
}

func sendAlertMail(alertInfo *AlertInfo, config *Config) {
	l4g.Info("Send mail: " + alertInfo.title + " : " + alertInfo.msg)
	ml := mail.NewMail(config.alertConfig.smtpServer, config.alertConfig.smtpPort,
		config.alertConfig.mailFrom, config.alertConfig.password)
	err := ml.Send(config.alertConfig.mailTo, alertInfo.title, alertInfo.msg)
	if err != nil {
		l4g.Error(err.Error())
	}
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
	config.alertConfig.smtpServer = alertConfig[CONF_ALERT_SMTP_SERVER].(string)
	config.alertConfig.smtpPort = alertConfig[CONF_ALERT_SMTP_PORT].(string)
	config.alertConfig.mailTo = alertConfig[CONF_ALERT_MAILTO].(string)
	config.alertConfig.mailFrom = alertConfig[CONF_ALERT_MAILFROM].(string)
	config.alertConfig.password = alertConfig[CONF_ALERT_PASSWORD].(string)
	config.alertConfig.tmplDstatDiskTitle = alertConfig[CONF_ALERT_TMPL_DSTAT_DISK_TITLE].(string)
	config.alertConfig.tmplDstatDiskMsg = alertConfig[CONF_ALERT_TMPL_DSTAT_DISK_MSG].(string)
	config.alertConfig.tmplDstatDiskNormalTitle = alertConfig[CONF_ALERT_TMPL_DSTAT_DISK_NORMAL_TITLE].(string)
	config.alertConfig.tmplDstatDiskNormalMsg = alertConfig[CONF_ALERT_TMPL_DSTAT_DISK_NORMAL_MSG].(string)
	config.alertConfig.tmplDstatCpuTitle = alertConfig[CONF_ALERT_TMPL_DSTAT_CPU_TITLE].(string)
	config.alertConfig.tmplDstatCpuMsg = alertConfig[CONF_ALERT_TMPL_DSTAT_CPU_MSG].(string)
	config.alertConfig.tmplDstatCpuNormalTitle = alertConfig[CONF_ALERT_TMPL_DSTAT_CPU_NORMAL_TITLE].(string)
	config.alertConfig.tmplDstatCpuNormalMsg = alertConfig[CONF_ALERT_TMPL_DSTAT_CPU_NORMAL_MSG].(string)
	config.alertConfig.tmplDstatMemTitle = alertConfig[CONF_ALERT_TMPL_DSTAT_MEM_TITLE].(string)
	config.alertConfig.tmplDstatMemMsg = alertConfig[CONF_ALERT_TMPL_DSTAT_MEM_MSG].(string)
	config.alertConfig.tmplDstatMemNormalTitle = alertConfig[CONF_ALERT_TMPL_DSTAT_MEM_NORMAL_TITLE].(string)
	config.alertConfig.tmplDstatMemNormalMsg = alertConfig[CONF_ALERT_TMPL_DSTAT_MEM_NORMAL_MSG].(string)

	dstatConfigs := m[CONF_DSTAT].([]interface{})
	config.dstatConfigs = make([]DstatConfig, len(dstatConfigs))
	for i, dstatConfig := range dstatConfigs {
		tmp := dstatConfig.(map[interface{}]interface{})
		config.dstatConfigs[i].server = tmp[CONF_DSTAT_SERVER].(string)
		config.dstatConfigs[i].cpurate = (int64)(tmp[CONF_DSTAT_CPU].(int))
		config.dstatConfigs[i].diskrate = (int64)(tmp[CONF_DSTAT_DISK].(int))
		config.dstatConfigs[i].memrate = (int64)(tmp[CONF_DSTAT_MEM].(int))
		config.dstatConfigs[i].interval = (int64)(tmp[CONF_DSTAT_INTERVAL].(int))
	}

	return config
}
