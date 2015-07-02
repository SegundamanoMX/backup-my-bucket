# Copyright 2015 ASM Clasificados de Mexico, SA de CV
#
# This file is part of backup-my-bucket.
#
# backup-my-bucket is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License Version 2 as published by
# the Free Software Foundation.
#
# backup-my-bucket is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with backup-my-bucket.  If not, see <http://www.gnu.org/licenses/>.

all: backup-my-bucket

backup-my-bucket: rpm-setuptree
	rpmbuild -bc backup-my-bucket.spec

rpm: rpm-setuptree
	rpmbuild -bb backup-my-bucket.spec 

rpm-setuptree:
	mkdir -p build/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

clean:
	rm -rf build
