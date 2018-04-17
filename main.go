package main

import (
	"flag"
	l4g "github.com/alecthomas/log4go"
	"github.com/yfujita/monitoring-elasticsearch-fluent/mail"
	"github.com/yfujita/monitoring-elasticsearch-fluent/monitor"
	"github.com/yfujita/slackutil"
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
	CONF_ALERT_SLACK_WEBHOOK_URL            = "slack-webhook-url"
	CONF_ALERT_SLACK_CHANNEL                = "slack-channel"
	CONF_ALERT_SLACK_BOT_NAME               = "slack-bot-name"
	CONF_ALERT_SLACK_BOT_ICON               = "slack-bot-icon"
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
	CONF_ALERT_TMPL_APPLOG_TITLE            = "tmpl-applog-title"
	CONF_ALERT_TMPL_APPLOG_MSG              = "tmpl-applog-msg"
	CONF_DSTAT                              = "dstat"
	CONF_DSTAT_SERVER                       = "server"
	CONF_DSTAT_CPU                          = "cpu-threshold"
	CONF_DSTAT_DISK                         = "disk-threshold"
	CONF_DSTAT_MEM                          = "mem-threshold"
	CONF_DSTAT_INTERVAL                     = "interval"
	CONF_APPLOG                             = "applog"
	CONF_APPLOG_SERVER                      = "server"
	CONF_APPLOG_LOGNAME                     = "logname"
	CONF_APPLOG_KEYWORD                     = "keyword"
	CONF_APPLOG_EXCLUDES                    = "excludes"
	CONF_APPLOG_INTERVAL                    = "interval"
	CONF_SCRIPT                             = "script"
	CONF_SCRIPT_DIR                         = "directory"
	CONF_SCRIPT_INTERVAL                    = "interval"
	STATE_GOOD                              = "good"
	STATE_WARING                            = "warning"
	STATE_DANGER                            = "danger"
	STATE_NONE                              = ""
)

type AlertInfo struct {
	title string
	msg   string
	state string
}

type Config struct {
	eshost        string
	esport        string
	alertConfig   AlertConfig
	dstatConfigs  []DstatConfig
	applogConfigs []ApplogConfig
	scriptConfig  *ScriptConfig
}

type AlertConfig struct {
	smtpServer               string
	smtpPort                 string
	mailTo                   string
	mailFrom                 string
	password                 string
	slackWebHookUrl          string
	slackChannel             string
	slackBotName             string
	slackBotIcon             string
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
	tmplApplogTitle          string
	tmplApplogMsg            string
}

type DstatConfig struct {
	server   string
	cpurate  int64
	diskrate int64
	memrate  int64
	interval int64
}

type ApplogConfig struct {
	server   string
	logname  string
	keyword  string
	excludes string
	interval int64
}

type ScriptConfig struct {
	scriptDir string
	interval  int64
}

func main() {
	var confPath string
	flag.StringVar(&confPath, "conf", "blank", "config file path")
	flag.Parse()
	confPath = path.Clean(confPath)
	if confPath == "blank" {
		panic("Invalid conf parameter")
	}

	l4g.LoadConfiguration(confPath + "/" + LOG4G_XML)
	defer l4g.Close()
	config := loadConfig(confPath + "/" + CONF_FILE)

	l4g.Info("start monitoring")

	alertCh := make(chan *AlertInfo, 100)

	if len(config.dstatConfigs) > 0 {
		for _, dstatConfig := range config.dstatConfigs {
			go monitoringDstat(config.eshost, config.esport, config.alertConfig, dstatConfig, alertCh)
		}
	}

	if len(config.applogConfigs) > 0 {
		for _, appConfig := range config.applogConfigs {
			go monitoringApplog(config.eshost, config.esport, config.alertConfig, appConfig, alertCh)
		}
	}

	if config.scriptConfig != nil {
		go monitorScript(config.alertConfig, config.scriptConfig, alertCh)
	}

	go func(ch chan *AlertInfo, config *Config) {
		for {
			ai := <-ch
			sendAlertMail(ai, config)
		}
	}(alertCh, config)

	for {
		//TODO
		time.Sleep(1 * time.Hour)
	}
	l4g.Info("finish")
}

func monitoringApplog(hostName, port string, alertConfig AlertConfig, applogConfig ApplogConfig, ch chan *AlertInfo) {
	l4g.Info("start applog monitoring task. logname:%s keyword:%s", applogConfig.logname, applogConfig.keyword)
	typeName := applogConfig.server
	timestamp := time.Now().Unix() * 1000
	ma := monitor.NewMonitorApplog(hostName, port, applogConfig.server)
	var alertTime int64 = 0
	for {
		l4g.Debug("start applog monitoring. logname:%s keyword:%s", applogConfig.logname, applogConfig.keyword)

		l4g.Debug("Applog timestamp from %d", timestamp)
		infos, err := ma.GetApplogInfo(applogConfig.logname, applogConfig.keyword, applogConfig.excludes, timestamp, 10)
		if err != nil {
			l4g.Error(err.Error())
			time.Sleep((time.Duration)(applogConfig.interval) * time.Second)
			continue
		}

		if len(infos) == 0 {
			time.Sleep((time.Duration)(applogConfig.interval) * time.Second)
			continue
		}

		for _, info := range infos {
			ai := NewAlertInfo(alertConfig.tmplApplogTitle, alertConfig.tmplApplogMsg,
				typeName, info.Message, STATE_DANGER)
			ai.title = strings.Replace(ai.title, "{logname}", info.LogName, -1)
			ai.msg = strings.Replace(ai.msg, "{logname}", info.LogName, -1)
			ai.title = strings.Replace(ai.title, "{keyword}", info.Keyword, -1)
			ai.msg = strings.Replace(ai.msg, "{keyword}", info.Keyword, -1)
			tmpTime, err := time.Parse("2006-01-02T15:04:05-07:00", info.Timestamp)
			if err != nil {
				tmpTime, err = time.Parse("2006-01-02T15:04:05.000Z", info.Timestamp)
			}
			if err == nil {
				timestamp = (tmpTime.Unix() + 1) * 1000
				l4g.Info("Next time = %d", timestamp)
			} else {
				l4g.Warn(err.Error())
			}
			if (timestamp - alertTime) > (30 * 60 * 1000) {
				ch <- ai
				alertTime = timestamp
			} else {
				l4g.Info("Skip alert: " + ai.title + " " + ai.msg)
			}
		}
		time.Sleep((time.Duration)(applogConfig.interval) * time.Second)

		//l4g.Info(ret)
	}
}

func monitoringDstat(hostName, port string, alertConfig AlertConfig, dstatConfig DstatConfig, ch chan *AlertInfo) {
	typeName := dstatConfig.server
	num := 12

	md := monitor.NewMonitorDstat(hostName, port, typeName)
	diskAlert := false
	cpuAlert := false
	memAlert := false

	l4g.Info("start dstat monitoring task. server:%s", typeName)

	for {
		l4g.Debug("start dstat monitoring. server:%s", typeName)

		timestamp := (time.Now().Unix() - (60 * 1000)) * 1000
		infos, err := md.GetDstatInfo(timestamp, num)
		if err != nil {
			l4g.Error(err.Error())
			time.Sleep((time.Duration)(dstatConfig.interval) * time.Second)
			continue
		}
		if len(infos) == 0 {
			l4g.Warn("%s dstat log doesnt exist.", typeName)
			time.Sleep((time.Duration)(dstatConfig.interval) * time.Second)
			continue
		}

		cpuRate, diskRate, memRate := GetResourceUsageRate(infos)

		if (int64)(diskRate) >= dstatConfig.diskrate && !diskAlert {
			diskAlert = true
			ch <- NewAlertInfo(alertConfig.tmplDstatDiskTitle, alertConfig.tmplDstatDiskMsg,
				typeName, strconv.FormatInt(diskRate, 10), STATE_WARING)
		} else if (int64)(diskRate) < dstatConfig.diskrate && diskAlert {
			diskAlert = false
			ch <- NewAlertInfo(alertConfig.tmplDstatDiskNormalTitle, alertConfig.tmplDstatDiskNormalMsg,
				typeName, strconv.FormatInt(diskRate, 10), STATE_GOOD)
		}

		if cpuRate >= dstatConfig.cpurate && !cpuAlert {
			cpuAlert = true
			ch <- NewAlertInfo(alertConfig.tmplDstatCpuTitle, alertConfig.tmplDstatCpuMsg,
				typeName, strconv.FormatInt(cpuRate, 10), STATE_WARING)
		} else if cpuRate < dstatConfig.cpurate && cpuAlert {
			cpuAlert = false
			ch <- NewAlertInfo(alertConfig.tmplDstatCpuNormalTitle, alertConfig.tmplDstatCpuNormalMsg,
				typeName, strconv.FormatInt(cpuRate, 10), STATE_GOOD)
		}

		if memRate >= dstatConfig.memrate && !memAlert {
			memAlert = true
			ch <- NewAlertInfo(alertConfig.tmplDstatMemTitle, alertConfig.tmplDstatMemMsg,
				typeName, strconv.FormatInt(memRate, 10), STATE_WARING)
		} else if memRate < dstatConfig.memrate && memAlert {
			memAlert = false
			ch <- NewAlertInfo(alertConfig.tmplDstatMemNormalTitle, alertConfig.tmplDstatMemNormalMsg,
				typeName, strconv.FormatInt(memRate, 10), STATE_GOOD)
		}
		time.Sleep((time.Duration)(dstatConfig.interval) * time.Second)
	}

}

func monitorScript(alertConfig AlertConfig, scriptConfig *ScriptConfig, ch chan *AlertInfo) {
	l4g.Info("monitorScript")
	ms := monitor.NewMonitorScript(scriptConfig.scriptDir)
	failureMap := map[string]int64{}
	for {
		results, err := ms.GetScriptResult()
		if err != nil {
			l4g.Error(err.Error())
			time.Sleep((time.Duration)(scriptConfig.interval) * time.Second)
			continue
		}

		for _, result := range results {
			if result.ExitCode == 0 {
				if failureMap[result.Filename] != 0 {
					ch <- NewAlertInfo("Monitoring Script back to normal. file:"+result.Filename+" exit:"+strconv.FormatInt(result.ExitCode, 10), result.SystemOut, "", "", STATE_GOOD)
				}
			} else {
				if failureMap[result.Filename] != result.ExitCode {
					l4g.Info("Monitoring Script Failure. file:" + result.Filename + " exit:" + strconv.FormatInt(result.ExitCode, 10))
					ch <- NewAlertInfo("Monitoring Script Failure. file:"+result.Filename+" exit:"+strconv.FormatInt(result.ExitCode, 10), result.SystemOut, "", "", STATE_DANGER)
				}
			}
			failureMap[result.Filename] = result.ExitCode
		}

		time.Sleep((time.Duration)(scriptConfig.interval) * time.Second)
	}
}

func GetResourceUsageRate(infos []*monitor.DstatInfo) (cpuRate, diskRate, memRate int64) {
	num := len(infos)
	if num == 0 {
		return 0, 0, 0
	}

	var diskRateTotal int64 = 0
	var cpuRateTotal int64 = 0
	var memRateTotal int64 = 0

	for _, info := range infos {
		diskRateTotal += info.DiskUsed * 100 / (info.DiskUsed + info.DiskFree)
		cpuRateTotal += info.CpuUsr + info.CpuSystem
		memRateTotal += info.MemUsed * 100 / (info.MemUsed + info.MemBuff + info.MemCach + info.MemFree)
	}

	diskRate = diskRateTotal / (int64)(num)
	cpuRate = cpuRateTotal / (int64)(num)
	memRate = memRateTotal / (int64)(num)
	return
}

func NewAlertInfo(titleTemplate, msgTemplate, server, value, state string) *AlertInfo {
	ai := new(AlertInfo)
	ai.title = strings.Replace(
		strings.Replace(titleTemplate, "{server}", server, -1),
		"{value}", value, -1)
	ai.msg = strings.Replace(
		strings.Replace(msgTemplate, "{server}", server, -1),
		"{value}", value, -1)
	ai.state = state
	return ai
}

func sendAlertMail(alertInfo *AlertInfo, config *Config) {
	if len(config.alertConfig.mailTo) > 0 {
		l4g.Info("Send mail: " + alertInfo.title + " : " + alertInfo.msg)
		ml := mail.NewMail(config.alertConfig.smtpServer, config.alertConfig.smtpPort,
			config.alertConfig.mailFrom, config.alertConfig.password)
		err := ml.Send(config.alertConfig.mailTo, alertInfo.title, alertInfo.msg)
		if err != nil {
			l4g.Error(err.Error())
		}
	}

	if len(config.alertConfig.slackWebHookUrl) > 0 {
		l4g.Info("Send slack: " + alertInfo.title + " : " + alertInfo.msg)
		bot := slackutil.NewBot(config.alertConfig.slackWebHookUrl, config.alertConfig.slackChannel,
			config.alertConfig.slackBotName, config.alertConfig.slackBotIcon)
		var err error
		if len(alertInfo.state) == 0 {
			err = bot.Message(alertInfo.title, alertInfo.msg)
		} else {
			attachments := make(map[string]string)
			attachments["title"] = alertInfo.title
			if STATE_DANGER == alertInfo.state {
				attachments["text"] = "@here: " + alertInfo.msg
			} else {
				attachments["text"] = alertInfo.msg
			}
			attachments["color"] = alertInfo.state
			err = bot.MessageWithAttachments("", []map[string]string{attachments})
		}

		if err != nil {
			l4g.Error(err.Error())
		}
	}
}

func loadConfig(path string) *Config {
	yaml, err := ioutil.ReadFile(path)

	m := make(map[interface{}]interface{})
	err = goyaml.Unmarshal(yaml, &m)
	if err != nil {
		panic(err.Error())
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
	config.alertConfig.slackWebHookUrl = alertConfig[CONF_ALERT_SLACK_WEBHOOK_URL].(string)
	config.alertConfig.slackChannel = alertConfig[CONF_ALERT_SLACK_CHANNEL].(string)
	config.alertConfig.slackBotName = alertConfig[CONF_ALERT_SLACK_BOT_NAME].(string)
	config.alertConfig.slackBotIcon = alertConfig[CONF_ALERT_SLACK_BOT_ICON].(string)
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
	config.alertConfig.tmplApplogTitle = alertConfig[CONF_ALERT_TMPL_APPLOG_TITLE].(string)
	config.alertConfig.tmplApplogMsg = alertConfig[CONF_ALERT_TMPL_APPLOG_MSG].(string)

	if m[CONF_DSTAT] != nil {
		dstatConfigs := m[CONF_DSTAT].([]interface{})
		config.dstatConfigs = make([]DstatConfig, len(dstatConfigs))
		if len(dstatConfigs) > 0 {
			for i, dstatConfig := range dstatConfigs {
				tmp := dstatConfig.(map[interface{}]interface{})
				config.dstatConfigs[i].server = tmp[CONF_DSTAT_SERVER].(string)
				config.dstatConfigs[i].cpurate = (int64)(tmp[CONF_DSTAT_CPU].(int))
				config.dstatConfigs[i].diskrate = (int64)(tmp[CONF_DSTAT_DISK].(int))
				config.dstatConfigs[i].memrate = (int64)(tmp[CONF_DSTAT_MEM].(int))
				config.dstatConfigs[i].interval = (int64)(tmp[CONF_DSTAT_INTERVAL].(int))
			}
		}
	}

	if m[CONF_APPLOG] != nil {
		applogConfigs := m[CONF_APPLOG].([]interface{})
		config.applogConfigs = make([]ApplogConfig, len(applogConfigs))
		if len(applogConfigs) > 0 {
			for i, applogConfig := range applogConfigs {
				tmp := applogConfig.(map[interface{}]interface{})
				config.applogConfigs[i].server = tmp[CONF_APPLOG_SERVER].(string)
				config.applogConfigs[i].logname = tmp[CONF_APPLOG_LOGNAME].(string)
				config.applogConfigs[i].keyword = tmp[CONF_APPLOG_KEYWORD].(string)
				config.applogConfigs[i].interval = (int64)(tmp[CONF_APPLOG_INTERVAL].(int))
				if tmp[CONF_APPLOG_EXCLUDES] == nil {
					config.applogConfigs[i].excludes = ""
				} else {
					config.applogConfigs[i].excludes = tmp[CONF_APPLOG_EXCLUDES].(string)
				}
			}
		}
	}

	if m[CONF_SCRIPT] != nil {
		scriptConfig := m[CONF_SCRIPT].(map[interface{}]interface{})
		config.scriptConfig = new(ScriptConfig)
		config.scriptConfig.scriptDir = scriptConfig[CONF_SCRIPT_DIR].(string)
		config.scriptConfig.interval = int64(scriptConfig[CONF_SCRIPT_INTERVAL].(int))
	}

	return config
}
