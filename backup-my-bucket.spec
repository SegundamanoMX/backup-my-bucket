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

Name:           backup-my-bucket
Version:        0.1.0
Release:        %(expr `git rev-list HEAD --count`)
Summary:        A tool for backup and restore of S3 buckets within AWS.
Group:          Applications/Archiving

License:        Copyright 2015 ASM Clasificados de Mexico, SA de CV
URL:            https://github.com/SegundamanoMX/backup-my-bucket
Source:         https://github.com/SegundamanoMX/backup-my-bucket

BuildRequires:  golang >= 1.3.3

%define _topdir %(pwd)/build
%define _src %(pwd)
%define _pkg src/github.com/SegundamanoMX/backup-my-bucket
%define bindir /opt/backup-my-bucket

%description
backup-my-bucket is a tool for backup and restore of S3 buckets within
AWS.  backup-my-bucket is written in Golang.

%prep
export GOPATH=%{_builddir}
go get -d github.com/vaughan0/go-ini github.com/aws/aws-sdk-go
mkdir -p %{_pkg}
cd %{_src} && cp -r *.go *.conf common gc log ls restore snapshot  %{_builddir}/%{_pkg}

%build
export GOPATH=%{_builddir}
cd %{_pkg} && go build

%pre
if ! id bambu &>/dev/null; then useradd bambu; fi

%install
install -d %{buildroot}%{bindir}
install -p -m 0755 %{_pkg}/backup-my-bucket %{buildroot}%{bindir}
install -p -m 0644 %{_pkg}/backup-my-bucket.conf %{buildroot}%{bindir}

%clean
rm -rf %{buildroot}

%files
%defattr(-,bambu,bambu,-)
%{bindir}/backup-my-bucket
%config(noreplace) %{bindir}/backup-my-bucket.conf
