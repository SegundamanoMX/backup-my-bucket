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

package log

import (
	"os"
	"fmt"
	"log"
	"log/syslog"
)

const (
	QUIET = 0
	INFO = 1
	DEBUG = 2
)

var (
	Info = func(format string, params ...interface{}) {}
	Debug = func(format string, params ...interface{}) {}
	Error = func(format string, params ...interface{}) {}
	Fatal = func(format string, params ...interface{}) {}
)

func Init(logToSyslog bool, logLevel int) {

	if logLevel == QUIET {
		Info = func(format string, params ...interface{}) {}
		Debug = func(format string, params ...interface{}) {}
		Error = func(format string, params ...interface{}) {}
		Fatal = func(format string, params ...interface{}) {}
		return
	}

	if logToSyslog {
		logSys, err := syslog.New(syslog.LOG_INFO|syslog.LOG_LOCAL0, "backup-my-bucket")
		if err != nil {
			error := log.New(os.Stdout, "ERROR ", log.Ldate|log.Lmicroseconds|log.Lshortfile)
                        error.Fatalf("Error starting syslog logger: %s", err)
                }
		Info = func(format string, params ...interface{}) {
			logSys.Info(fmt.Sprintf(format, params...))
		}
		if logLevel >= DEBUG {
			Debug = func(format string, params ...interface{}) {
				logSys.Debug(fmt.Sprintf(format, params...))
			}
		}
		Error = func(format string, params ...interface{}) {
			logSys.Err(fmt.Sprintf(format, params...))
		}
		Fatal = func(format string, params ...interface{}) {
			logSys.Crit("FATAL " + fmt.Sprintf(format, params...))
			logSys.Close()
			os.Exit(1)
		}
	} else {
		info := log.New(os.Stderr, "INFO ", log.Ldate|log.Lmicroseconds)
		debug := log.New(os.Stderr, "DEBUG ", log.Ldate|log.Lmicroseconds)
		error := log.New(os.Stderr, "ERROR ", log.Ldate|log.Lmicroseconds)
		fatal := log.New(os.Stderr, "FATAL ", log.Ldate|log.Lmicroseconds)
		Info = info.Printf
		if logLevel >= DEBUG { Debug = debug.Printf }
		Error = error.Printf
		Fatal = fatal.Fatalf
	}
}
