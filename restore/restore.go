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

package restore

import (
	"bytes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/SegundamanoMX/backup-my-bucket/common"
	"github.com/SegundamanoMX/backup-my-bucket/log"
	"io/ioutil"
)

type DownloadWork struct {
	Wid                  int
	Version              common.Version
	Retry                int
}

type UploadWork struct {
	Wid                  int
	Version              common.Version
	Bytes                []byte
	Retry                int
}

var (
	readyRestoreWorkers                = make(chan int, common.RestoreWorkerCount)
	downloadWorkQueue                  = make(chan DownloadWork, common.RestoreWorkerCount)
	uploadWorkQueue                    = make(chan UploadWork, common.RestoreWorkerCount)
)

func Restore(snapshotName string) {
	snapshot := common.LoadSnapshot(common.Cfg.BackupSet.SnapshotsDir + snapshotName)

	log.Info("Restoring bucket %s to snapshot %s.", common.Cfg.BackupSet.MasterBucket, snapshotName)

	common.ConfigureAws(common.Cfg.BackupSet.MasterRegion)

	for i := 0; i < common.RestoreWorkerCount; i++ {
		readyRestoreWorkers <- i
		go downloadWorker()
		go uploadWorker()
	}

	for _, version := range snapshot.Contents {
		wid := <-readyRestoreWorkers
		downloadWorkQueue <- DownloadWork{Wid: wid, Version: version, Retry: 0}
	}

	for i := common.RestoreWorkerCount; i > 0; i-- {
		log.Info("Wait for %d restore workers to finish.", i)
		wid := <-readyRestoreWorkers
		log.Info("Restore worker [%d] finished.", wid)
	}
	close(downloadWorkQueue)
	close(uploadWorkQueue)

	log.Info("Restored bucket %s to snapshot %s.", common.Cfg.BackupSet.MasterBucket, snapshotName)
}

func downloadWorker() {

	s3Client := s3.New(nil)

	for work := range downloadWorkQueue {

		log.Debug("[%d] Download version, retry %d: %s", work.Wid, work.Retry, work.Version)
		getParams := &s3.GetObjectInput{
			Bucket:                     aws.String(common.Cfg.BackupSet.SlaveBucket), // Required
			Key:                        aws.String(work.Version.Key),  // Required
			// IfMatch:                    aws.String("IfMatch"),
			// IfModifiedSince:            aws.Time(time.Now()),
			// IfNoneMatch:                aws.String("IfNoneMatch"),
			// IfUnmodifiedSince:          aws.Time(time.Now()),
			// Range:                      aws.String("Range"),
			// RequestPayer:               aws.String("RequestPayer"),
			// ResponseCacheControl:       aws.String("ResponseCacheControl"),
			// ResponseContentDisposition: aws.String("ResponseContentDisposition"),
			// ResponseContentEncoding:    aws.String("ResponseContentEncoding"),
			// ResponseContentLanguage:    aws.String("ResponseContentLanguage"),
			// ResponseContentType:        aws.String("ResponseContentType"),
			// ResponseExpires:            aws.Time(time.Now()),
			// SSECustomerAlgorithm:       aws.String("SSECustomerAlgorithm"),
			// SSECustomerKey:             aws.String("SSECustomerKey"),
			// SSECustomerKeyMD5:          aws.String("SSECustomerKeyMD5"),
			VersionId:                  aws.String(work.Version.VersionId),
		}
		getResp, getErr := s3Client.GetObject(getParams)
		if getErr != nil {
			if awsErr, ok := getErr.(awserr.Error); ok {
				log.Error("[%d] Error code '%s', message '%s', origin '%s'", work.Wid, awsErr.Code(), awsErr.Message(), awsErr.OrigErr())
				if reqErr, ok := getErr.(awserr.RequestFailure); ok {
					log.Error("[%d] Service error code '%s', message '%s', status code '%d', request id '%s'", work.Wid, reqErr.Code(), reqErr.Message(), reqErr.StatusCode(), reqErr.RequestID())
				}
			} else {
				// This case should never be hit, The SDK should alwsy return an
				// error which satisfies the awserr.Error interface.
				log.Error("[%d] Non AWS error: %s", work.Wid, getErr.Error())
			}

			work.Retry++
			if work.Retry == common.MaxRetries {
				log.Fatal("[%d] Error downloading version, retry %d: %s", work.Wid, work.Retry, work.Version)
			}
			log.Error("[%d] Error downloading version, retry %d: %s", work.Wid, work.Retry, work.Version)
			downloadWorkQueue <- work
			continue
		}

		log.Debug("[%d] Read response: %s", work.Wid, work.Version)
		bytes, readErr := ioutil.ReadAll(getResp.Body)
		getResp.Body.Close()
		if readErr != nil {
			work.Retry++
			if work.Retry == common.MaxRetries {
				log.Fatal("[%d] Could not read version %s, retry %d: %s", work.Wid, work.Version, work.Retry, readErr)
			}
			log.Error("[%d] Could not read version %s, retry %d: %s", work.Wid, work.Version, work.Retry, readErr)
			downloadWorkQueue <- work
			continue
		}

		log.Debug("[%d] Downloaded version: %s", work.Wid, work.Version)
		uploadWorkQueue <- UploadWork{Wid: work.Wid, Version: work.Version, Bytes: bytes, Retry: 0}
	}
}

func uploadWorker() {

	s3Client := s3.New(nil)

	for work := range uploadWorkQueue {

		log.Debug("[%d] Upload version, retry %d: %s", work.Wid, work.Retry, work.Version)
		putParams := &s3.PutObjectInput{
			Bucket:             aws.String(common.Cfg.BackupSet.MasterBucket), // Required
			Key:                aws.String(work.Version.Key),  // Required
			// ACL:                aws.String("ObjectCannedACL"),
			Body:               bytes.NewReader(work.Bytes),
			// CacheControl:       aws.String("CacheControl"),
			// ContentDisposition: aws.String("ContentDisposition"),
			// ContentEncoding:    aws.String("ContentEncoding"),
			// ContentLanguage:    aws.String("ContentLanguage"),
			// ContentLength:      aws.Long(1),
			// ContentType:        aws.String("ContentType"),
			// Expires:            aws.Time(time.Now()),
			// GrantFullControl:   aws.String("GrantFullControl"),
			// GrantRead:          aws.String("GrantRead"),
			// GrantReadACP:       aws.String("GrantReadACP"),
			// GrantWriteACP:      aws.String("GrantWriteACP"),
			// Metadata: map[string]*string{
			// 	"Key": aws.String("MetadataValue"), // Required
			//	// More values...
			// },
			// RequestPayer:            aws.String("RequestPayer"),
			// SSECustomerAlgorithm:    aws.String("SSECustomerAlgorithm"),
			// SSECustomerKey:          aws.String("SSECustomerKey"),
			// SSECustomerKeyMD5:       aws.String("SSECustomerKeyMD5"),
			// SSEKMSKeyID:             aws.String("SSEKMSKeyId"),
			// ServerSideEncryption:    aws.String("ServerSideEncryption"),
			// StorageClass:            aws.String("StorageClass"),
			// WebsiteRedirectLocation: aws.String("WebsiteRedirectLocation"),
		}
		_, putErr := s3Client.PutObject(putParams)

		if putErr != nil {
			if awsErr, ok := putErr.(awserr.Error); ok {
				log.Error("[%d] Error code '%s', message '%s', origin '%s'", work.Wid, awsErr.Code(), awsErr.Message(), awsErr.OrigErr())
				if reqErr, ok := putErr.(awserr.RequestFailure); ok {
					log.Error("[%d] Service error code '%s', message '%s', status code '%d', request id '%s'", work.Wid, reqErr.Code(), reqErr.Message(), reqErr.StatusCode(), reqErr.RequestID())
				}
			} else {
				// This case should never be hit, The SDK should alwsy return an
				// error which satisfies the awserr.Error interface.
				log.Error("[%d] Non AWS error: %s", work.Wid, putErr.Error())
			}

			work.Retry++
			if work.Retry == common.MaxRetries {
				log.Fatal("[%d] Error uploading version, retry %d: %s", work.Wid, work.Retry, work.Version)
			}
			log.Error("[%d] Error uploading version, retry %d: %s", work.Wid, work.Retry, work.Version)
			uploadWorkQueue <- work
			continue
		}

		log.Info("[%d] Restored version: %s", work.Wid, work.Version)
		readyRestoreWorkers <- work.Wid
	}
}
