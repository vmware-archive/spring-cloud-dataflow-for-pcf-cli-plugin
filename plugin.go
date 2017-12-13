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

	"os/exec"

	"io"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/cfutil"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/cli"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/dataflow"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/download"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/download/cache"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/format"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/httpclient"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/java"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/pluginutil"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/serviceutil"
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
	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse // avoid following redirects
		},
	}
	authClient := httpclient.NewAuthenticatedClient(client)

	argsConsumer := cli.NewArgConsumer(args, diagnoseWithHelp)

	switch args[0] {

	case "dataflow-shell":
		dataflowSIName := getDataflowServerInstanceName(argsConsumer)

		runAction(argsConsumer, cliConnection, fmt.Sprintf("Attaching shell to dataflow service %s", format.Bold(format.Cyan(dataflowSIName))), func(progressWriter io.Writer) (string, error) {
			argsConsumer.CheckAllConsumed()
			accessToken, err := cfutil.GetToken(cliConnection)
			if err != nil {
				return "", err
			}

			dataflowServer, err := serviceutil.ServiceInstanceURL(cliConnection, dataflowSIName, accessToken, authClient)
			if err != nil {
				return "", err
			}

			return "", downloadAndRunShell(func() (string, string, error) {
				return dataflow.DataflowShellDownloadUrl(dataflowServer, authClient)
			}, func(fileName string) *exec.Cmd {
				return dataflow.DataflowShellCommand(fileName, dataflowServer, skipSslValidation)
			}, progressWriter)
		})

	default:
		os.Exit(0) // Ignore CLI-MESSAGE-UNINSTALL etc.
	}
}

type urlResolver func() (string, string, error)

type shellCommandFactory func(fileName string) *exec.Cmd

func downloadAndRunShell(shellDownloadUrl urlResolver, shellCommand shellCommandFactory, progressWriter io.Writer) error {
	url, checksum, err := shellDownloadUrl()
	if err != nil {
		return err
	}

	downloadCache, err := cache.NewCache()
	if err != nil {
		return err
	}
	httpHelper := download.NewHttpHelper()
	downloader, err := download.NewDownloader(downloadCache, httpHelper)
	if err != nil {
		return err
	}

	filePath, err := downloader.DownloadFile(url, checksum)
	if err != nil {
		return err
	}

	fmt.Fprintln(progressWriter, "Launching dataflow shell JAR")
	err = java.RunShell(shellCommand(filePath))
	if err != nil {
		fmt.Fprintln(progressWriter, "Launching dataflow shell JAR failed. Checking java installation")
		checkErr := java.Check(progressWriter, err)
		if checkErr != nil {
			return fmt.Errorf("java is needed. Please install a JRE or JDK and try again. Details: %s", checkErr.Error())
		}
		fmt.Fprintln(progressWriter, "java installation appears to be ok. Any error messages above may indicate why launching dataflow shell JAR failed.")

	}
	return err
}

func getDataflowServerInstanceName(ac *cli.ArgConsumer) string {
	return ac.Consume(1, "dataflow server service instance name")
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
					Usage: "   cf dataflow-shell DATAFLOW_SERVER_SERVICE_INSTANCE_NAME",
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

func runAction(argsConsumer *cli.ArgConsumer, cliConnection plugin.CliConnection, message string, action func(progressWriter io.Writer) (string, error)) {
	argsConsumer.CheckAllConsumed()

	format.RunAction(cliConnection, message, action, os.Stdout, func() {
		os.Exit(1)
	})
}
