monitoring-elasticsearch-fluent
===============================

monitoring log of fluent-plugin-elasticsearch

    go run main.go -conf ./config

### td-agent.conf for dstat
reference -> http://blog.nomadscafe.jp/2014/03/dstat-fluentd-elasticsearch-kibana.html
```xml:td-agent.conf
<source>
  type dstat
  tag dstat
  option -lcnm --freespace
  delay 10
</source>

<match dstat>
  type copy
  <store>
    type map
    tag  "map.dstat.loadavg-short"
    time time
    record {"value" => record["dstat"]["load avg"]["1m"], "stat" => "loadavg-short", "host" => record["hostname"]}
  </store>
  <store>
    type map
    tag  "map.dstat.cpu-usr"
    time time
    record {"value" => record["dstat"]["total cpu usage"]["usr"], "stat" => "cpu-usr", "host" => record["hostname"]}
  </store>
  <store>
    type map
    tag  "map.dstat.cpu-sys"
    time time
    record {"value" => record["dstat"]["total cpu usage"]["sys"], "stat" => "cpu-sys", "host" => record["hostname"]}
  </store>
  <store>
    type map
    tag  "map.dstat.cpu-hiq"
    time time
    record {"value" => record["dstat"]["total cpu usage"]["hiq"], "stat" => "cpu-hiq", "host" => record["hostname"]}
  </store>
  <store>
    type map
    tag  "map.dstat.cpu-siq"
    time time
    record {"value" => record["dstat"]["total cpu usage"]["siq"], "stat" => "cpu-siq", "host" => record["hostname"]}
  </store>
  <store>
    type map
    tag  "map.dstat.net-recv"
    time time
    record {"value" => record["dstat"]["net/total"]["recv"], "stat" => "net-recv", "host" => record["hostname"]}
  </store>  
  <store>
    type map
    tag  "map.dstat.net-send"
    time time
    record {"value" => record["dstat"]["net/total"]["send"], "stat" => "net-send", "host" => record["hostname"]}
  </store>  
  <store>
    type map
    tag  "map.dstat.disk-used"
    time time
    record {"value" => record["dstat"]["/"]["used"], "stat" => "disk-used", "host" => record["hostname"]}
  </store>
  <store>
    type map
    tag  "map.dstat.disk-free"
    time time
    record {"value" => record["dstat"]["/"]["free"], "stat" => "disk-free", "host" => record["hostname"]}
  </store>
  <store>
    type map
    tag  "map.dstat.mem-used"
    time time
    record {"value" => record["dstat"]["memory usage"]["used"], "stat" => "mem-used", "host" => record["hostname"]}
  </store>
  <store>
    type map
    tag  "map.dstat.mem-buff"
    time time
    record {"value" => record["dstat"]["memory usage"]["buff"], "stat" => "mem-buff", "host" => record["hostname"]}
  </store>
  <store>
    type map
    tag  "map.dstat.mem-cach"
    time time
    record {"value" => record["dstat"]["memory usage"]["cach"], "stat" => "mem-cach", "host" => record["hostname"]}
  </store>
  <store>
    type map
    tag  "map.dstat.mem-free"
    time time
    record {"value" => record["dstat"]["memory usage"]["free"], "stat" => "mem-free", "host" => record["hostname"]}
  </store>
</match>

<match map.dstat.*>
  type elasticsearch
  type_name       serverName
  host            localhost
  port            9200
  logstash_format true
  logstash_prefix dstat
  flush_interval  30s
</match>
```