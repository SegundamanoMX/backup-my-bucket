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

package ls

import (
	"path/filepath"
	"fmt"
	"github.com/SegundamanoMX/backup-my-bucket/common"
	"github.com/SegundamanoMX/backup-my-bucket/log"
)

func ListSnapshots() {
	log.Info("Listing snapshots for backup set.")
	snapshots := common.LoadSnapshots()
	fmt.Println("Snapshot                         Timestamp                        Key count          Total size")
	fmt.Println("-----------------------------------------------------------------------------------------------")
	for _, snapshot := range snapshots {
		var size int64 = 0
		for _, version := range snapshot.Contents {
			size =+ version.Size
		}
		fmt.Printf ("%-33s%-33s%-15d%12dKb\n", filepath.Base(snapshot.File), snapshot.Timestamp.Format("2006-01-02 15:04:05 -0700 MST"), len(snapshot.Contents), size)
	}
}

