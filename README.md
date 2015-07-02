# backup-my-bucket

backup-my-bucket is a command line application that complements
cross-region replication and versioning to form a backup and restore
system for S3 buckets. For a given master bucket, the system covers
the following use cases.

1. Create restoration points.
2. Restore the contents of master to a restoration point.
3. Remove obsolete restoration points.

A restoration point is a copy of the contents of master at a given
point in time. The system creates restoration points by copying the
contents of master into a given slave bucket and then recording the
current keys in the local filesystem. The copy of master to slave
happens continually by means of cross-region replication and
versioning. Cross-region replication continually
[replicates](http://docs.aws.amazon.com/AmazonS3/latest/dev/crr-what-is-isnot-replicated.html)
the state of the master bucket in the slave bucket. Versioning
[archives](http://docs.aws.amazon.com/AmazonS3/latest/dev/AddingObjectstoVersioningEnabledBuckets.html)
the current state of the slave bucket when a new state is
replicated. Creation of the index happens when you run command
`backup-my-bucket snapshot`. The command snapshots the contents
of slave bucket and stores the index in a snapshot file.

Restoring master consists in copying the contents of a restoration
point from the slave bucket to the master bucket. Restoration happens
when you run command `backup-my-bucket restore <SNAPSHOT>`. The
command copies from slave the key versions indicated by `SNAPSHOT`.

A restoration point is obsolete when it's older than the retention
policy. Removal of obsolete restoration points happens when you run
command `backup-my-bucket gc`. For a given obsolete restoration point,
the command will remove the corresponding snapshot and versions.

## Compile

On systems with `go`, `make`, and `rpmbuild`, you can fetch
dependencies and build the program like so.

```
backup-my-bucket$ make
```

On any other system, fetch dependencies with comand

```
go get -d github.com/vaughan0/go-ini github.com/aws/aws-sdk-go
```

then build by [`go
build`](http://golang.org/pkg/go/build/).

## Install

We provide a RPM package that you create
by means of Make. You build the RPM like so.

```
backup-my-bucket$ make rpm
```

The installation directory is `/opt/backup-my-bucket/`.

## Configure

After [installation](#install), you will find a boilerplate
configuration file in
`/opt/backup-my-bucket/backup-my-bucket.conf`. The configuration file
is a JSON consisting of the following fields.

- `LogLevel`: Log level of backup-my-bucket. Possible values are
  `0` (quiet), `1` (info), and `2` (debug).
- `AwsLogLevel`: Log level of the [aws sdk for
  go](http://godoc.org/github.com/aws/aws-sdk-go). AFAIS by consulting the
  [source code](https://github.com/aws/aws-sdk-go), possible
  values are `0` (quiet) and `1` (debug).
- `Syslog`: Switch between logging to stderr (value `false`) and logging to
  syslog (value `true`). When logging to syslog, backup-my-bucket will
  log to facility `local0.info` with tag `backup-my-bucket`.
- `BackupSet`: The one and only backupset. We may or may not support
  multiple backupsets in the future.
  - `SnapshotsDir`: Directory where backup-my-bucket will store
    snapshots.
  - `CompressSnapshots`: Switch between storing subsequent snapshots
    as plaintext files (value `false`) and as compressed files (value
    `true`).
  - `MinimumRedundancy`: Safety parameter that indicates the minimum
    count of restoration points that backup-my-bucket
    keeps. backup-my-bucket removes a restoration point only when
    there are `MinimumRedundancy + 1` restoration points.
  - `RetentionPolicy`: Age limit in days for restoration points. Older
    restoration points are considered obsolete and thus removed by command
    `backup-my-bucket gc`.
  - `MasterBucket`: Name of master bucket.
  - `MasterRegion`: Name of region of master bucket as given by
    [Amazon
    AWS](http://docs.aws.amazon.com/general/latest/gr/rande.html).
  - `SlaveBucket`, `SlaveRegion`: Values for slave bucket
    corresponding to previous two parameters.
  - `AccessKey`: Amazon AWS access key id.
  - `SecretKey`: Amazon AWS secret access key.

## Create restoration point

Create a restoration point by copying the contents of your master
bucket to a versioned slave bucket and then snapshot the slave
bucket. You guarantee that the slave bucket is versioned and
up-to-date by enabling cross-region replication between your master
and slave buckets. After copy, create snapshot the slave bucket with
command `backup-my-bucket snapshot`. This will store corresponding
snapshot file locally as indicated by your
[configuration](#configure). You may want to schedule a cron job
for running the command periodically.

## List restoration points

Run command `backup-my-bucket list-snapshots`.

## Restore master bucket

Run command `backup-my-bucket restore <SNAPSHOT>`.

## Remove obsolete snapshots

Run command `backup-my-bucket gc`. For a given obsolete restoration point,
the command will remove the corresponding snapshot and versions. The
command will not remove an obsolete restoration point when doing so
reduces the count of restoration points bellow the [minimum redundancy
parameter](#configure).

## Limitations

1. Copy files from master to slave during creation of restoration
   points. Instead, we rely on cross region replication to copy
   contents in advance.
2. Compress contents of restoration points.
3. Delete any given restoration point.
4. Switch master and slave roles for disaster recovery by failover.

## Copyright

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
