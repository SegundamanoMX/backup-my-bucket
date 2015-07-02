/*
Copyright 2015 ASM Clasificados de Mexico, SA de CV

This file is part of backup-my-bucket.

backup-my-bucket is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License Version 2 as published by
the Free Software Foundation.

backup-my-bucket is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with backup-my-bucket.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/SegundamanoMX/backup-my-bucket/common"
	"github.com/SegundamanoMX/backup-my-bucket/gc"
	"github.com/SegundamanoMX/backup-my-bucket/ls"
	"github.com/SegundamanoMX/backup-my-bucket/log"
	"github.com/SegundamanoMX/backup-my-bucket/restore"
	"github.com/SegundamanoMX/backup-my-bucket/snapshot"
	"io/ioutil"
	"os"
	"runtime"
)

var (
	configFile           *string
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	parseParams()
	loadConfig()
	log.Init(common.Cfg.Syslog, common.Cfg.LogLevel)
	for i, param := range flag.Args() {
		switch param {
		case "snapshot":
			snapshot.Snapshot()
			return
		case "list-snapshots":
			ls.ListSnapshots()
			return
		case "restore":
			snapshotName := flag.Args()[i+1:]
			if len(snapshotName) == 1 {
				restore.Restore(snapshotName[0])
				return
			} else {
				log.Fatal("Too many or too few parameters for command restore: %s", snapshotName)
			}
		case "gc":
			gc.GarbageCollect()
			return
		default:
			log.Fatal("Found unhandled command '%s'.", param)
		}
	}
	log.Fatal("No command was given.")
}

func parseParams() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: backup-my-bucket [-help] [-config] {snapshot,list-snapshots,restore,gc}:\n")
		fmt.Fprintf(os.Stderr, "commands:\n")
		fmt.Fprintf(os.Stderr, "  snapshot:                          Create a restoration point\n")
		fmt.Fprintf(os.Stderr, "  list-snapshots:                    List available restoration points\n")
		fmt.Fprintf(os.Stderr, "  restore:                           Restore master bucket at given restoration point\n")
		fmt.Fprintf(os.Stderr, "  gc:                                Garbage collect obsolete restoration points\n")
		fmt.Fprintf(os.Stderr, "optional arguments:\n")
		flag.PrintDefaults()
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("FATAL: Could not query current directory.")
		os.Exit(1)
	}
	configFile = flag.String("config", cwd + "/backup-my-bucket.conf", "Path to configuration file")

	flag.Parse()
}

func loadConfig()  {
	bytes, err := ioutil.ReadFile(*configFile)
	if err != nil {
		fmt.Println("ERROR in configuration file '%s'", *configFile)
		fmt.Println(err)
		os.Exit(1)
	}
	err = json.Unmarshal(bytes, &common.Cfg)
	if err != nil {
		fmt.Println("ERROR in configuration file '%s'", *configFile)
		fmt.Println(err)
		os.Exit(1)
	}
}
