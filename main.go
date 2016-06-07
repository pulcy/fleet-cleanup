// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/juju/errgo"
	"github.com/op/go-logging"
	"github.com/spf13/cobra"

	"github.com/pulcy/fleet-cleanup/service"
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"

	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	projectName = "fleet-cleanup"

	defaultLogLevel = "debug"
	defaultEtcdAddr = "http://localhost:2379"
)

type globalOptions struct {
	logLevel string
	etcdAddr string
	dryRun   bool
}

var (
	cmdMain = &cobra.Command{
		Use: projectName,
		Run: cmdMainRun,
	}
	globalFlags globalOptions
)

func init() {
	logging.SetFormatter(logging.MustStringFormatter("[%{level:-5s}] %{message}"))

	cmdMain.Flags().StringVar(&globalFlags.logLevel, "log-level", defaultLogLevel, "Minimum log level (debug|info|warning|error)")
	cmdMain.Flags().StringVar(&globalFlags.etcdAddr, "etcd-addr", defaultEtcdAddr, "Address of etcd")
	cmdMain.Flags().BoolVar(&globalFlags.dryRun, "dry-run", false, "If set, only list garbage, but do not remove it")
}

func main() {
	cmdMain.Execute()
}

func cmdMainRun(cmd *cobra.Command, args []string) {
	// Parse arguments
	if globalFlags.etcdAddr == "" {
		Exitf("Please specify --etcd-addr")
	}
	etcdUrl, err := url.Parse(globalFlags.etcdAddr)
	if err != nil {
		Exitf("--etcd-addr '%s' is not valid: %#v", globalFlags.etcdAddr, err)
	}

	// Set log level
	setLogLevel(globalFlags.logLevel, projectName)

	// Update service config (if needed)
	serviceLogger := logging.MustGetLogger(projectName)
	service, err := service.NewService(service.ServiceConfig{
		EtcdURL: *etcdUrl,
		DryRun:  globalFlags.dryRun,
	}, service.ServiceDependencies{
		Logger: serviceLogger,
	})
	if err != nil {
		Exitf("Failed to create service: %#v", err)
	}

	if err := service.Run(); err != nil {
		Exitf("Failed to run service: %#v", err)
	}
}

func showUsage(cmd *cobra.Command, args []string) {
	cmd.Usage()
}

func Exitf(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}
	fmt.Printf(format, args...)
	fmt.Println()
	os.Exit(1)
}

func assert(err error) {
	if err != nil {
		Exitf("Assertion failed: %#v", err)
	}
}

func assertArgIsSet(arg, argKey string) {
	if arg == "" {
		Exitf("%s must be set\n", argKey)
	}
}

func setLogLevel(logLevel, loggerName string) {
	level, err := logging.LogLevel(logLevel)
	if err != nil {
		Exitf("Invalid log-level '%s': %#v", logLevel, err)
	}
	logging.SetLevel(level, loggerName)
}
