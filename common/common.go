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

package common

import (
	"compress/gzip"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/SegundamanoMX/backup-my-bucket/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type BackupSet struct {
	SnapshotsDir         string
	CompressSnapshots    bool
	MinimumRedundancy    int
	RetentionPolicy      int
	MasterBucket         string
	MasterRegion         string
	SlaveBucket          string
	SlaveRegion          string
	AccessKey            string
	SecretKey            string
}

type AppConfig struct {
	LogLevel             int
	AwsLogLevel          aws.LogLevelType
	Syslog               bool
	BackupSet            BackupSet
}

type Version struct {
	Key                  string
	LastModified         time.Time
	Size                 int64
	VersionId            string
}

type Snapshot struct {
	File                 string
	Timestamp            time.Time `json:"Timestamp"`
	Contents             []Version `json:"Contents"`
}

const (
	SnapshotWorkerCount  = 128
	SnapshotBatchSize    = 100000
	RestoreWorkerCount   = 1024
	MaxRetries           = 10
	GcBatchSize          = 1
)

var (
	Cfg                  AppConfig
)

func LoadSnapshots() (snapshots []Snapshot) {
	log.Info("Loading snapshots")
	files, _ := filepath.Glob(Cfg.BackupSet.SnapshotsDir + "/*")
	for _, file := range files {
		snapshots = append(snapshots, LoadSnapshot(file))
	}
	return
}

func LoadSnapshot(file string) (snapshot Snapshot) {
	log.Info("Loading snapshot file '%s'.", file)
	var bytes []byte
	var readErr error
	if filepath.Ext(file) == ".Z" {
		f, openErr := os.OpenFile(file, os.O_RDONLY, 0000)
		if openErr != nil {
			log.Fatal("Could not open file %s: %s", file, openErr)
		}
		defer f.Close()

		r, nrErr := gzip.NewReader(f)
		if nrErr != nil {
			log.Fatal("Could not initialize gzip decompressor: %s", nrErr)
		}
		defer r.Close()

		bytes, readErr = ioutil.ReadAll(r)
		if readErr != nil {
			log.Fatal("Could not read compressed snapshot file %s: %s", file, readErr)
		}
	} else {
		bytes, readErr = ioutil.ReadFile(file)
		if readErr != nil {
			log.Fatal("Could not read snapshot file '%s': %s", file, readErr)
		}
	}
	err := json.Unmarshal(bytes, &snapshot)
	if err != nil {
		log.Fatal("Could not parse snapshot file '%s': %s", file, err)
	}
	snapshot.File = file
	if Cfg.LogLevel > 0 {
		pretty, _ := json.MarshalIndent(snapshot, "", "    ")
		log.Debug("Snapshot '%s':\n%s", file, pretty)
	}
	return
}

func ConfigureAws(region string) {
	logLevelType := func(logLevel uint) aws.LogLevelType {
		var awsLogLevel aws.LogLevelType
		switch logLevel {
		case 0: awsLogLevel = aws.LogOff
		case 1: awsLogLevel = aws.LogDebug
		case 2: awsLogLevel = aws.LogDebugWithSigning
		case 3: awsLogLevel = aws.LogDebugWithHTTPBody
		case 4: awsLogLevel = aws.LogDebugWithRequestRetries
		case 5: awsLogLevel = aws.LogDebugWithRequestErrors
		}
		return awsLogLevel
	}

	defaults.DefaultConfig.Credentials = credentials.NewStaticCredentials(Cfg.BackupSet.AccessKey, Cfg.BackupSet.SecretKey, "")
	defaults.DefaultConfig.Region = &region
	defaults.DefaultConfig.LogLevel = aws.LogLevel(logLevelType(uint(Cfg.AwsLogLevel)))
}

func Min(x int, y int) int {
	if (x < y) {
		return x
	} else {
		return y
	}
}
