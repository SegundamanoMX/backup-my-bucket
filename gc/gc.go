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

package gc

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/SegundamanoMX/backup-my-bucket/common"
	"github.com/SegundamanoMX/backup-my-bucket/log"
	"math"
	"os"
	"time"
)

func GarbageCollect() {
	log.Info("Garbage collecting obsolete backups.")
	snapshots := common.LoadSnapshots()
	if (len(snapshots) <= common.Cfg.BackupSet.MinimumRedundancy) {
		log.Fatal("Minimum redundancy is not met. Current snapshot count is %d.", len(snapshots))
	}
	oldSnapshots, recentSnapshots := discriminateSnapshots(snapshots)
	discriminateVersions(oldSnapshots, recentSnapshots)
	versionsToRemove := discriminateVersions(oldSnapshots, recentSnapshots)
	if ok := removeVersions(versionsToRemove); !ok {
		log.Fatal("There was an unhandled error removing obsolete versions, exiting.")
	}
	removeSnapshots(oldSnapshots)
}

func discriminateSnapshots(snapshots []common.Snapshot) (old []common.Snapshot, recent []common.Snapshot) {
	retentionPeriod := time.Now().AddDate(0, 0, - common.Cfg.BackupSet.RetentionPolicy)
	log.Info("Retention period is from %s up until now.", retentionPeriod)
	for _, snapshot := range snapshots {
		if retentionPeriod.After(snapshot.Timestamp) {
			log.Info("Snapshot '%s' on %s is old.", snapshot.File, snapshot.Timestamp)
			old = append(old, snapshot)
		} else {
			log.Info("Snapshot '%s' on %s is recent.", snapshot.File, snapshot.Timestamp)
			recent = append(recent, snapshot)
		}
	}
	return
}

func discriminateVersions(oldSnapshots []common.Snapshot, recentSnapshots []common.Snapshot) (versionsToRemove []common.Version) {
	recentId := make(map[string]bool)

	for _, recentS := range recentSnapshots {
		log.Debug("Collecting versions for recent snapshot '%s' on %s.", recentS.File, recentS.Timestamp)
		for _, recentVersion := range recentS.Contents {
			if common.Cfg.LogLevel >= log.DEBUG {
				pretty, _ := json.MarshalIndent(recentVersion, "", "    ")
				log.Debug("Version is recent: %s", pretty)
			}
			recentId[recentVersion.VersionId] = true;
		}
	}	

	for _, oldS := range oldSnapshots {
		log.Debug("Discriminating versions for old Snapshot '%s' on %s.", oldS.File, oldS.Timestamp)
		for _, oldVersion := range oldS.Contents {
			if _, ok := recentId[oldVersion.VersionId]; ok {
				continue
			}
			recentId[oldVersion.VersionId] = false
			if common.Cfg.LogLevel > log.DEBUG {
				pretty, _ := json.MarshalIndent(oldVersion, "", "    ")
				log.Debug("Will remove version: %s", pretty)
			}
			versionsToRemove = append(versionsToRemove, oldVersion)
		}
	}
	return
}

func removeVersions(versionsToRemove []common.Version) (ok bool) {
	objectBatches := makeObjectBatches(versionsToRemove)

	common.ConfigureAws(common.Cfg.BackupSet.SlaveRegion)

	s3Client := s3.New(nil)

	for batch, objects := range objectBatches {
		params := &s3.DeleteObjectsInput{
			Bucket: aws.String(common.Cfg.BackupSet.SlaveBucket),
			Delete: &s3.Delete{
				Objects: objects,
				Quiet: aws.Bool(true),
			},
		}

		log.Debug("params[%d] = %+v", batch, *params)
		resp, err := s3Client.DeleteObjects(params)

		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ! ok {
				// According to aws-sdk-go documentation:
				// This case should never be hit, The SDK should alwsy return an
				// error which satisfies the awserr.Error interface.
				log.Error(err.Error())
				return false
			}

			// Generic AWS Error with Code, Message, and original error (if any)
			log.Error(awsErr.Code(), awsErr.Message(), awsErr.OrigErr())
			if reqErr, ok := err.(awserr.RequestFailure); ok {
				// A service error occurred
				log.Error(reqErr.Code(), reqErr.Message(), reqErr.StatusCode(), reqErr.RequestID())
			}

			if awsErr.Code() != "NoSuchVersion" {
				return false
			}
		}

		log.Debug("response[%d] = %+v", batch, *resp)
	}
	return true
}

func makeObjectBatches(versions []common.Version) (objectBatches [][]*s3.ObjectIdentifier) {
	objectBatches = make([][]*s3.ObjectIdentifier, int(math.Ceil(float64(len(versions)) / float64(common.GcBatchSize))))

	batch := 0
	for count := common.Min(len(versions), common.GcBatchSize); count <= len(versions); count += common.GcBatchSize {
		objectBatches[batch] = make([]*s3.ObjectIdentifier, common.Min(common.GcBatchSize, count - batch * common.GcBatchSize))
		batch++
	}

	batch = -1
	for index, version := range versions {
		if index % common.GcBatchSize == 0 {
			batch++
		}
		object := &s3.ObjectIdentifier{
			Key:       aws.String(version.Key),
			VersionId: aws.String(version.VersionId),
		}
		objectBatches[batch][index - common.GcBatchSize * batch] = object
		log.Debug("Batch %d = %+v", batch, objectBatches[batch])
	}

	return
}

func removeSnapshots(snapshots []common.Snapshot) {
	for _, snapshot := range snapshots {
		if err := os.Remove(snapshot.File); err != nil {
			log.Error("Error removing snapshot '%s': %s", snapshot.File, err)
		}
	}
}
