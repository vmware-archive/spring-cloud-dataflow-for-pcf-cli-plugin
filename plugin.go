/*
 * Copyright 2017-Present the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/cli"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/format"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/httpclient"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/pluginutil"
	"os/exec"
	"strings"
	"io"
)

// Plugin version. Substitute "<major>.<minor>.<build>" at build time, e.g. using -ldflags='-X main.pluginVersion=1.2.3'
var pluginVersion = "invalid version - plugin was not built correctly"

// Plugin is a struct implementing the Plugin interface, defined by the core CLI, which can
// be found in "code.cloudfoundry.org/cli/plugin/plugin.go".
type Plugin struct{}

func (c *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
	skipSslValidation, err := cliConnection.IsSSLDisabled()
	if err != nil {
		format.Diagnose(string(err.Error()), os.Stderr, func() {
			os.Exit(1)
		})
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSslValidation},
	}
	client := &http.Client{Transport: tr}
	authClient := httpclient.NewAuthenticatedClient(client)

	argsConsumer := cli.NewArgConsumer(args, diagnoseWithHelp)

	switch args[0] {

	case "dataflow-shell":
		configServerInstanceName := getDataflowServerInstanceName(argsConsumer)
		_ = configServerInstanceName
		_ = authClient

		// Prototype implementation:
		url := "https://repo.spring.io/libs-snapshot/org/springframework/cloud/spring-cloud-dataflow-shell/1.2.3.RELEASE/spring-cloud-dataflow-shell-1.2.3.RELEASE.jar"
		var fileName string
		{
			tokens := strings.Split(url, "/")
			fileName = tokens[len(tokens)-1]

			_, err := os.Stat(fileName)
			if err != nil {
				fmt.Printf("Downloading %s\n", url)
				file, err := os.Create(fileName)
				if err != nil {
					fmt.Printf("Error creating %s: %s\n", fileName, err)
					return
				}
				defer file.Close()

				response, err := http.Get(url)
				if err != nil {
					fmt.Printf("Error accessing %s: %s\n", url, err)
					return
				}
				defer response.Body.Close()

				_, err = io.Copy(file, response.Body)
				if err != nil {
					fmt.Printf("Error downloading %s: %s\n", url, err)
					return
				}
			}

		}

		cmd := exec.Command("java", "-jar", fileName, "--dataflow.uri=http://localhost:9393/")
		cmd.Env = []string{fmt.Sprintf("PATH=%s", os.Getenv("PATH"))}

		stdin, err := cmd.StdinPipe()
		if err != nil {
			fmt.Printf("Error accessing shell's standard input pipe: %s\n", err)
		}
		defer stdin.Close()

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		go func() {
			io.Copy(stdin, os.Stdin)
		}()

		err = cmd.Run()
		if err != nil {
			fmt.Printf("Failed: %s\n", err)
			return
		}

	default:
		os.Exit(0) // Ignore CLI-MESSAGE-UNINSTALL etc.
	}
}

func getDataflowServerInstanceName(ac *cli.ArgConsumer) string {
	return ac.Consume(1, "dataflow server instance name")
}

func getServiceInstanceName(ac *cli.ArgConsumer) string {
	return ac.Consume(1, "service instance name")
}

func diagnoseWithHelp(message string, command string) {
	fmt.Printf("%s See 'cf help %s'.\n", message, command)
	os.Exit(1)
}

func failInstallation(format string, inserts ...interface{}) {
	// There is currently no way to emit the message to the command line during plugin installation. Standard output and error are swallowed.
	fmt.Printf(format, inserts...)
	fmt.Println("")

	// Fail the installation
	os.Exit(64)
}

func (c *Plugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:    "spring-cloud-dataflow-for-pcf-cli-plugin",
		Version: pluginutil.ParsePluginVersion(pluginVersion, failInstallation),
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 7,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "dataflow-shell",
				HelpText: "Open a dataflow shell to a Spring Cloud Dataflow for PCF dataflow server",
				Alias:    "dfsh",
				UsageDetails: plugin.Usage{
					Usage: "   cf dataflow-shell DATAFLOW_SERVER_INSTANCE_NAME",
				},
			},
		},
	}
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("This program is a plugin which expects to be installed into the cf CLI. It is not intended to be run stand-alone.")
		pv := pluginutil.ParsePluginVersion(pluginVersion, failInstallation)
		fmt.Printf("Plugin version: %d.%d.%d\n", pv.Major, pv.Minor, pv.Build)
		os.Exit(0)
	}
	plugin.Start(new(Plugin))
}
