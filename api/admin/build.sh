#!/bin/sh

set -x

ORPA_HOME=/root/go/pythonrpa/api/admin
cd $ORPA_HOME/tenant
go build

#scp /root/go/pythonrpa/api/admin/tenant/tenant root@10.164.39.165:/home/minikube/go/orpa/orpa/api

#/usr/local/bin/sshpass -p rpaLinux12. scp -r go/pythonrpa/api/admin/tenant/tenant root@10.164.39.165:/home/minikube/go/orpa/orpa/api

cd $ORPA_HOME/environment
go build

cd $ORPA_HOME/schedule
go build

cd $ORPA_HOME/bot
go build

cd $ORPA_HOME/user
go build

#cd $ORPA_HOME/worker
#go build

#cd $ORPA_HOME/sendlog
#go build
