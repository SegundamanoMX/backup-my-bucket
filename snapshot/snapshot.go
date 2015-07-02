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

package snapshot

import (
	"compress/gzip"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/SegundamanoMX/backup-my-bucket/common"
	"github.com/SegundamanoMX/backup-my-bucket/log"
	"io/ioutil"
	"time"
	"os"
)

var (
	readySnapshotWorkers              = make(chan int, common.SnapshotWorkerCount)
	doneSnapshotWorkers               = make(chan int, common.SnapshotWorkerCount)
	workRequests                      = make(chan string)
	snapshotWorkQueue                 = make([]string, 0)
	versionsFunnel                    = make(chan []common.Version, common.SnapshotWorkerCount)
	versions                          = make([]common.Version, 0)
)

func Snapshot() {

	timestamp := time.Now()
	timestampStr := timestamp.Format("20060102150405-0700MST")
	log.Info("Taking snapshot %s of bucket %s.", timestamp, common.Cfg.BackupSet.SlaveBucket)
	log.Info("Taking snapshot %s of bucket %s.", timestampStr, common.Cfg.BackupSet.SlaveBucket)
	
	common.ConfigureAws(common.Cfg.BackupSet.SlaveRegion)
	for wid := 0; wid < common.SnapshotWorkerCount; wid++ {
		readySnapshotWorkers <- wid
	}

	go func (){ workRequests <- "" }()
	go dispatchWorkers()

	for newVersions := range versionsFunnel {
		versions = append(versions, newVersions...)
	}

	log.Info("Dumping snapshot to %s%s.", common.Cfg.BackupSet.SnapshotsDir, timestampStr)
	snapshot := &common.Snapshot{
		File: common.Cfg.BackupSet.SnapshotsDir + timestampStr,
		Timestamp: timestamp,
		Contents: versions,
	}
	if common.Cfg.BackupSet.CompressSnapshots { snapshot.File += ".Z" }
	bytes, err := json.MarshalIndent(snapshot, "", "    ")
	if err != nil {
		log.Fatal("Could not marshal snapshot %s: %s", timestampStr, err)
	}
	if common.Cfg.BackupSet.CompressSnapshots {
		f, openErr := os.OpenFile(snapshot.File, os.O_WRONLY|os.O_CREATE, 0644)
		if openErr != nil {
			log.Fatal("Could not open file %s: %s", snapshot.File, openErr)
		}
		defer f.Close()

		w := gzip.NewWriter(f)
		if _, writeErr := w.Write(bytes); writeErr != nil {
			log.Fatal("Could not write compressed snapshot file %s: %s", snapshot.File, writeErr)
		}
		w.Close()
	} else {
		if writeErr := ioutil.WriteFile(snapshot.File, bytes, 0644); err != nil {
			log.Fatal("Could not write snapshot file %s: %s", snapshot.File, writeErr)
		}
	}
	log.Info("Snapshot %s of bucket %s is DONE.", timestampStr, common.Cfg.BackupSet.SlaveBucket)
}

func dispatchWorkers() {
	forloop: for {
		select {
		case path := <-workRequests:
			select {
			case wid := <-readySnapshotWorkers:
				go snapshotWorker(wid, path)
			default:
				snapshotWorkQueue = append(snapshotWorkQueue, path)
			}
		case wid := <- doneSnapshotWorkers:
			if len(snapshotWorkQueue) == 0 { break forloop }
			path := snapshotWorkQueue[0]
			snapshotWorkQueue = snapshotWorkQueue[1:]
			go snapshotWorker(wid, path)
		}
	}

	for i := common.SnapshotWorkerCount; i > 1; i-- {
		log.Info("Wait for %d snapshot workers to finish.", i)
		select {
		case wid := <-readySnapshotWorkers:
			log.Info("Snapshot worker [%d] finished, was never executed.", wid)
		case wid := <-doneSnapshotWorkers:
			log.Info("Snapshot worker [%d] finished.", wid)
		}
	}
	log.Info("All snapshot workers finished.")
	close(versionsFunnel)
}

func snapshotWorker(wid int, path string) {

	log.Info("[%d] Explore path '%s'.", wid, path)
	
	s3Client := s3.New(nil)
	params := &s3.ListObjectVersionsInput{
		Bucket:          aws.String(common.Cfg.BackupSet.SlaveBucket),
		Delimiter:       aws.String("/"),
		// EncodingType:    aws.String("EncodingType"),
		// KeyMarker:       aws.String("KeyMarker"),
		MaxKeys:         aws.Long(common.SnapshotBatchSize),
		Prefix:          aws.String(path),
		// VersionIDMarker: aws.String("VersionIdMarker"),
	}
	var discoveredVersions []common.Version
	buffer := make([]common.Version, common.SnapshotBatchSize)

	for batch := 1; ; batch++{
		log.Debug("[%d] Request batch %d for path '%s'", wid, batch, path)
		resp, err := s3Client.ListObjectVersions(params)
		
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if reqErr, ok := err.(awserr.RequestFailure); ok {
					// A service error occurred
					log.Error(awsErr.Code(), awsErr.Message(), awsErr.OrigErr())
					log.Fatal(reqErr.Code(), reqErr.Message(), reqErr.StatusCode(), reqErr.RequestID())
				} else {
					log.Fatal(awsErr.Code(), awsErr.Message(), awsErr.OrigErr())
				}
			} else {
				// This case should never be hit, The SDK should alway return an
				// error which satisfies the awserr.Error interface.
				log.Fatal(err.Error())
			}
		}

		for _, cp := range resp.CommonPrefixes {
			discoveredPath := *cp.Prefix
			log.Info("[%d] Discover path '%s'.", wid, discoveredPath)
			workRequests <- discoveredPath
		}

		index := 0
		for _, v := range resp.Versions {
			if v.IsLatest == nil { log.Fatal("[%d] IsLatest is nil", wid) }
			if v.IsLatest != nil && *v.IsLatest == true {
				if v.Key == nil { log.Fatal("[%d] Key is nil", wid) }
				if v.LastModified == nil { log.Fatal("[%d] LastModified is nil", wid) }
				if v.Size == nil { log.Fatal("[%d] Size is nil", wid) }
				if v.VersionID == nil { log.Fatal("[%d] VersionID is nil", wid) }
				version := common.Version{
					Key: *v.Key,
					LastModified: *v.LastModified,
					Size: *v.Size,
					VersionID: *v.VersionID,
				}
				log.Debug("[%d] Discover latest version: %s", wid, version)
				buffer[index] = version
				index++
			} else {
				log.Debug("[%d] Discover noncurrent latest version for key '%s'.", wid, *v.Key)
			}
		}
		discoveredVersions = append(discoveredVersions, buffer[0:index]...)

		if ! *resp.IsTruncated { break }
		log.Info("[%d] Continue exploring path '%s'.", wid, path)

		if resp.NextVersionIDMarker != nil {
			log.Debug("[%d] NextVersionIDMarker = %s", wid, awsutil.StringValue(resp.NextVersionIDMarker))
		} else {
			log.Debug("[%d] NextVersionIDMarker = nil", wid)
		}
		if resp.NextKeyMarker != nil {
			log.Debug("[%d] NextKeyMarker = %s", wid, awsutil.StringValue(resp.NextKeyMarker))
		} else {
			log.Debug("[%d] NextKeyMarker = nil", wid)
		}
		params.VersionIDMarker = resp.NextVersionIDMarker
		params.KeyMarker = resp.NextKeyMarker
	}

	log.Info("[%d] Registering versions for path '%s'.", wid, path)
	versionsFunnel <- discoveredVersions

	log.Info("[%d] Done exploring path '%s'.", wid, path)
	doneSnapshotWorkers <- wid
}
