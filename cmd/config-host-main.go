/*
 * Minio Client (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

var (
	configHostFlags = []cli.Flag{}
)

var configHostCmd = cli.Command{
	Name:   "host",
	Usage:  "List, modify and remove hosts in configuration file.",
	Flags:  append(configHostFlags, globalFlags...),
	Action: mainConfigHost,
	CustomHelpTemplate: `NAME:
   mc config {{.Name}} - {{.Usage}}

USAGE:
   mc config {{.Name}} OPERATION

OPERATION:
   add ALIAS URL ACCESS-KEY SECRET-KEY [API]
   remove ALIAS
   list

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Add Amazon S3 storage service under "myphotos" alias. For security reasons turn off bash history momentarily.
      $ set +o history
      $ mc config {{.Name}} add myphotos https://s3.amazonaws.com \
                  BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
      $ set -o history

   2. Add Amazon S3 accelerated storage service under "accel" alias. For security reasons turn off bash history momentarily.
      $ set +o history
      $ mc config {{.Name}} add accel https://s3-accelerate.amazonaws.com \
                  BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
      $ set -o history

   3. List all hosts.
      $ mc config {{.Name}} list

   4. Remove "goodisk" config.
      $ mc config {{.Name}} remove goodisk
`,
}

// hostMessage container for content message structure
type hostMessage struct {
	op        string
	Status    string `json:"status"`
	Alias     string `json:"alias"`
	URL       string `json:"URL"`
	AccessKey string `json:"accessKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty"`
	API       string `json:"api,omitempty"`
}

// String colorized host message
func (h hostMessage) String() string {
	switch h.op {
	case "list":
		message := console.Colorize("Alias", fmt.Sprintf("%s: ", h.Alias))
		message += console.Colorize("URL", fmt.Sprintf("%-30.30s", h.URL))
		if h.AccessKey != "" || h.SecretKey != "" {
			message += console.Colorize("AccessKey", fmt.Sprintf("  %-20.20s", h.AccessKey))
			message += console.Colorize("SecretKey", fmt.Sprintf("  %-40.40s", h.SecretKey))
			message += console.Colorize("API", fmt.Sprintf("  %.20s", h.API))
		}
		return message
	case "remove":
		return console.Colorize("HostMessage", "Removed ‘"+h.Alias+"’ successfully.")
	case "add":
		return console.Colorize("HostMessage", "Added ‘"+h.Alias+"’ successfully.")
	default:
		return ""
	}
}

// JSON jsonified host message
func (h hostMessage) JSON() string {
	h.Status = "success"
	jsonMessageBytes, e := json.Marshal(h)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// Validate command-line input args.
func checkConfigHostSyntax(ctx *cli.Context) {
	// show help if nothing is set
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "host", 1) // last argument is exit code
	}

	switch strings.TrimSpace(ctx.Args().First()) {
	case "add":
		checkConfigHostAddSyntax(ctx)
	case "remove":
		checkConfigHostRemoveSyntax(ctx)
	case "list":
	default:
		cli.ShowCommandHelpAndExit(ctx, "host", 1) // last argument is exit code
	}
}

// checkConfigHostAddSyntax - verifies input arguments to 'config host add'.
func checkConfigHostAddSyntax(ctx *cli.Context) {
	tailArgs := ctx.Args().Tail()
	tailsArgsNr := len(tailArgs)
	if tailsArgsNr < 4 || tailsArgsNr > 5 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for host add command.")
	}

	alias := tailArgs.Get(0)
	url := tailArgs.Get(1)
	accessKey := tailArgs.Get(2)
	secretKey := tailArgs.Get(3)
	api := tailArgs.Get(4)

	if !isValidAlias(alias) {
		fatalIf(errDummy().Trace(alias), "Invalid alias ‘"+alias+"’.")
	}

	if !isValidHostURL(url) {
		fatalIf(errDummy().Trace(url),
			"Invalid URL ‘"+url+"’.")
	}

	if !isValidAccessKey(accessKey) {
		fatalIf(errInvalidArgument().Trace(accessKey),
			"Invalid access key ‘"+accessKey+"’.")
	}

	if !isValidSecretKey(secretKey) {
		fatalIf(errInvalidArgument().Trace(secretKey),
			"Invalid secret key ‘"+secretKey+"’.")
	}

	if api != "" && !isValidAPI(api) { // Empty value set to default "S3v4".
		fatalIf(errInvalidArgument().Trace(api),
			"Unrecognized API signature. Valid options are ‘[S3v4, S3v2]’.")
	}
}

// checkConfigHostRemoveSyntax - verifies input arguments to 'config host remove'.
func checkConfigHostRemoveSyntax(ctx *cli.Context) {
	tailArgs := ctx.Args().Tail()

	if len(ctx.Args().Tail()) != 1 {
		fatalIf(errInvalidArgument().Trace(tailArgs...),
			"Incorrect number of arguments for remove host command.")
	}

	if !isValidAlias(tailArgs.Get(0)) {
		fatalIf(errDummy().Trace(tailArgs.Get(0)),
			"Invalid alias ‘"+tailArgs.Get(0)+"’.")
	}
}

func mainConfigHost(ctx *cli.Context) error {
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	// check 'config host' cli arguments.
	checkConfigHostSyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("HostMessage", color.New(color.FgGreen))
	console.SetColor("Alias", color.New(color.FgCyan, color.Bold))
	console.SetColor("URL", color.New(color.FgCyan))
	console.SetColor("AccessKey", color.New(color.FgBlue))
	console.SetColor("SecretKey", color.New(color.FgBlue))
	console.SetColor("API", color.New(color.FgYellow))

	cmd := ctx.Args().First()
	args := ctx.Args().Tail()

	// Switch case through commands.
	switch strings.TrimSpace(cmd) {
	case "add":
		alias := args.Get(0)
		url := args.Get(1)
		accessKey := args.Get(2)
		secretKey := args.Get(3)
		api := args.Get(4)
		if api == "" {
			api = "S3v4"
		}
		hostCfg := hostConfigV8{
			URL:       url,
			AccessKey: accessKey,
			SecretKey: secretKey,
			API:       api,
		}
		addHost(alias, hostCfg) // Add a host with specified credentials.
	case "remove":
		alias := args.Get(0)
		removeHost(alias) // Remove a host.
	case "list":
		listHosts() // List all configured hosts.
	}
	return nil
}

// addHost - add a host config.
func addHost(alias string, hostCfgV8 hostConfigV8) {
	mcCfgV8, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config ‘"+mustGetMcConfigPath()+"’.")

	// Add new host.
	mcCfgV8.Hosts[alias] = hostCfgV8

	err = saveMcConfig(mcCfgV8)
	fatalIf(err.Trace(alias), "Unable to update hosts in config version ‘"+mustGetMcConfigPath()+"’.")

	printMsg(hostMessage{
		op:        "add",
		Alias:     alias,
		URL:       hostCfgV8.URL,
		AccessKey: hostCfgV8.AccessKey,
		SecretKey: hostCfgV8.SecretKey,
		API:       hostCfgV8.API,
	})
}

// removeHost - removes a host.
func removeHost(alias string) {
	conf, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config version ‘"+globalMCConfigVersion+"’.")

	// Remove host.
	delete(conf.Hosts, alias)

	err = saveMcConfig(conf)
	fatalIf(err.Trace(alias), "Unable to save deleted hosts in config version ‘"+globalMCConfigVersion+"’.")

	printMsg(hostMessage{op: "remove", Alias: alias})
}

//byAlias is a collection satisfying sort.Interface
type byAlias []hostMessage

func (d byAlias) Len() int           { return len(d) }
func (d byAlias) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d byAlias) Less(i, j int) bool { return d[i].Alias < d[j].Alias }

// listHosts - list all host URLs.
func listHosts() {
	conf, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config version ‘"+globalMCConfigVersion+"’.")

	var maxAlias = 0
	for k := range conf.Hosts {
		if len(k) > maxAlias {
			maxAlias = len(k)
		}
	}
  
	var hosts []hostMessage
	for k, v := range conf.Hosts {
		hosts = append(hosts, hostMessage{
			op:        "list",
			Alias:     k,
			URL:       v.URL,
			AccessKey: v.AccessKey,
			SecretKey: v.SecretKey,
			API:       v.API,
		})
	}
  
  // Sort hosts by alias names lexically.
	sort.Sort(byAlias(hosts))
  
  // Display all the hosts.
	for _, host := range hosts {
    // Format properly for alignment based on alias length.
		host.Alias = fmt.Sprintf("%-*.*s", maxAlias, maxAlias, host.Alias)
		printMsg(host)
	}
}
