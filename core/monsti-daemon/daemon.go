// This file is part of Monsti, a web content management system.
// Copyright 2012-2013 Christian Neumann
//
// Monsti is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option) any
// later version.
//
// Monsti is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR
// A PARTICULAR PURPOSE.  See the GNU Affero General Public License for more
// details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Monsti.  If not, see <http://www.gnu.org/licenses/>.

/*
 Monsti is a simple and resource efficient CMS.

 This package implements the main daemon which starts and observes modules.
*/
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"pkg.monsti.org/monsti/api/service"
	"pkg.monsti.org/monsti/api/util"

	"pkg.monsti.org/gettext"
	"pkg.monsti.org/monsti/api/util/template"
)

// Settings for the application and the sites.
type settings struct {
	Monsti util.MonstiSettings
	// Listen is the host and port to listen for incoming HTTP connections.
	Listen string
	// List of modules to be activated.
	Modules []string
	Config  *Config
	Mail    struct {
		Host     string
		Username string
		Password string
		Debug    bool
	}
}

// moduleLog is a Writer used to log module messages on stderr.
type moduleLog struct {
	Type string
	Log  *log.Logger
}

func (s moduleLog) Write(p []byte) (int, error) {
	parts := bytes.SplitAfter(p, []byte("\n"))
	for _, part := range parts {
		if len(part) > 0 {
			s.Log.Print(s.Type, ": ", string(part))
		}
	}
	return len(p), nil
}

func main() {
	useSyslog := flag.Bool("syslog", false, "use syslog")

	flag.Parse()

	var logger *log.Logger
	if *useSyslog {
		var err error
		logger, err = syslog.NewLogger(syslog.LOG_INFO|syslog.LOG_DAEMON, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not setup syslog logger: %v\n", err)
			os.Exit(1)
		}
	} else {
		logger = log.New(os.Stderr, "monsti ", log.LstdFlags)
	}

	// Load configuration
	if flag.NArg() != 1 {
		logger.Fatalf("Usage: %v <config_directory>\n",
			filepath.Base(os.Args[0]))
	}
	cfgPath := util.GetConfigPath(flag.Arg(0))
	var settings settings
	if err := util.LoadModuleSettings("daemon", cfgPath, &settings); err != nil {
		logger.Fatal("Could not load settings: ", err)
	}

	var err error
	if settings.Config, err = loadConfig(filepath.Join(cfgPath, "conf.d")); err != nil {
		logger.Fatalf("Could not load application configuration: %v", err)
	}

	if err := (&settings).Monsti.LoadSiteSettings(); err != nil {
		logger.Fatal("Could not load site settings: ", err)
	}

	gettext.DefaultLocales.Domain = "monsti-daemon"
	gettext.DefaultLocales.LocaleDir = settings.Monsti.Directories.Locale

	var waitGroup sync.WaitGroup

	// Start service handler
	logger.Println("Setting up service")
	monstiPath := settings.Monsti.GetServicePath(service.MonstiService.String())
	monsti := new(MonstiService)
	monsti.Settings = &settings
	monsti.Logger = logger
	provider := service.NewProvider("Monsti", monsti)
	provider.Logger = logger
	if err := provider.Listen(monstiPath); err != nil {
		logger.Fatalf("service: Could not start service: %v", err)
	}
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		if err := provider.Accept(); err != nil {
			logger.Fatalf("Could not accept at service: %v", err)
		}
	}()

	// Start modules
	for _, module := range settings.Modules {
		logger.Println("Starting module", module)
		executable := "monsti-" + module
		cmd := exec.Command(executable, cfgPath)
		cmd.Stderr = moduleLog{module, logger}
		go func() {
			if err := cmd.Run(); err != nil {
				logger.Fatalf("Module %q failed: %v", module, err)
			}
		}()
	}

	// Setup up httpd
	handler := nodeHandler{
		Renderer: template.Renderer{Root: settings.Monsti.GetTemplatesPath()},
		Settings: &settings,
		Log:      logger,
		Sessions: service.NewSessionPool(1, monstiPath),
	}
	http.Handle("/static/", http.FileServer(http.Dir(
		filepath.Dir(settings.Monsti.GetStaticsPath()))))
	handler.Hosts = make(map[string]string)
	for site_title, site := range settings.Monsti.Sites {
		for _, host := range site.Hosts {
			handler.Hosts[host] = site_title
			http.Handle(host+"/site-static/", http.FileServer(http.Dir(
				filepath.Dir(settings.Monsti.GetSiteStaticsPath(site_title)))))
		}
	}
	http.Handle("/", &handler)
	waitGroup.Add(1)
	go func() {
		if err := http.ListenAndServe(settings.Listen, nil); err != nil {
			logger.Fatal("HTTP Listener failed: ", err)
		}
		waitGroup.Done()
	}()

	logger.Printf("Monsti is up and running, listening on %q", settings.Listen)
	waitGroup.Wait()
	logger.Println("Monsti is shutting down.")
}
