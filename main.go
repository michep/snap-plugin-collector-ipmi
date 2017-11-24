package main

import (
	"./ipmi"
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
)

func main() {
	plugin.StartCollector(ipmi.NewCollector(), ipmi.PluginName, ipmi.PluginVersion)
}
