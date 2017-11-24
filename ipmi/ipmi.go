package ipmi

import (
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	PluginVedor   = "mfms"
	PluginName    = "ipmi"
	PluginVersion = 1
	ipmiPath      = "ipmi_path"
	sudo          = "sudo"
)

type Plugin struct {
	initialized bool
	ipmiPath    string
	sudo        bool
}

type parseResult struct {
	name    string
	reading interface{}
	state   string
}

func NewCollector() *Plugin {
	return &Plugin{initialized: false}
}

func (p *Plugin) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	policy := plugin.NewConfigPolicy()
	policy.AddNewStringRule([]string{PluginVedor, PluginName}, ipmiPath, true, plugin.SetDefaultString("ipmitool"))
	policy.AddNewBoolRule([]string{PluginVedor, PluginName}, sudo, true, plugin.SetDefaultBool(true))
	return *policy, nil
}

func (p *Plugin) GetMetricTypes(plugin.Config) ([]plugin.Metric, error) {
	var mts []plugin.Metric
	namespace := createNamespace("state")
	mts = append(mts, plugin.Metric{Namespace: namespace})

	namespace = createNamespace("reading")
	mts = append(mts, plugin.Metric{Namespace: namespace})

	return mts, nil
}

func (p *Plugin) CollectMetrics(metrics []plugin.Metric) ([]plugin.Metric, error) {
	var err error
	var mts []plugin.Metric
	var results []parseResult
	var cmd *exec.Cmd

	if !p.initialized {
		p.ipmiPath, _ = metrics[0].Config.GetString(ipmiPath)
		p.sudo, _ = metrics[0].Config.GetBool(sudo)
		p.initialized = true
	}

	if p.sudo {
		cmd = exec.Command("sudo", p.ipmiPath, "sdr")
	} else {
		cmd = exec.Command(p.ipmiPath, "sdr")
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	outlines := strings.Split(string(output), "\n")
	var reading interface{}
	for _, line := range outlines {
		result := parseResult{}
		fields := smap(strings.Split(line, "|"), strings.TrimSpace)
		if len(fields) == 3 && fields[1] != "no reading" && fields[2] != "ns" {

			if strings.HasPrefix(fields[1], "0x") {
				reading, err = strconv.ParseInt(strings.TrimPrefix(strings.Fields(fields[1])[0], "0x"), 16, 64)
				if err != nil {
					continue
				}
			} else {
				reading, err = strconv.ParseInt(strings.Fields(fields[1])[0], 10, 64)
				if err != nil {
					reading, err = strconv.ParseFloat(strings.Fields(fields[1])[0], 64)
					if err != nil {
						continue
					}
				}
			}
			result.name = fields[0]
			result.reading = reading
			result.state = fields[2]
			results = append(results, result)
		}
	}

	ts := time.Now()
	for _, metric := range metrics {
		switch metric.Namespace[len(metric.Namespace)-1].Value {
		case "state":
			for _, result := range results {
				mt := plugin.Metric{
					Namespace: createNamespace("state"),
					Timestamp: ts,
				}
				mt.Namespace[2].Value = result.name
				if mt.Data = 1; result.state == "ok" {
					mt.Data = 0
				}
				mts = append(mts, mt)
			}
		case "reading":
			for _, result := range results {
				mt := plugin.Metric{
					Namespace: createNamespace("reading"),
					Timestamp: ts,
				}
				mt.Namespace[2].Value = result.name
				mt.Data = result.reading
				mts = append(mts, mt)
			}
		}
	}

	return mts, nil
}

func createNamespace(lastelement string) plugin.Namespace {
	namespace := plugin.NewNamespace(PluginVedor, PluginName)
	namespace = namespace.AddDynamicElement("sensor", "sensor name")
	namespace = namespace.AddStaticElement(lastelement)
	return namespace
}

func smap(in []string, f func(string) string) []string {
	res := make([]string, len(in))
	for i, s := range in {
		res[i] = f(s)
	}
	return res
}
