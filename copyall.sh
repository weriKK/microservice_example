#!/bin/bash
export GOOS=linux
export CGO_ENABLED=0

cd accountservice;go get;go build -o accountservice-linux-amd64;echo built `pwd`;cd ..
cd vipservice;go get;go build -o vipservice-linux-amd64;echo built `pwd`;cd ..
cd healthchecker;go get;go build -o healthchecker-linux-amd64;echo built `pwd`;cd ..

cp healthchecker/healthchecker-linux-amd64 accountservice/
cp healthchecker/healthchecker-linux-amd64 vipservice/


docker build -t kova/accountservice accountservice/
docker service rm accountservice
docker service create --log-driver=gelf --log-opt gelf-address=udp://192.168.1.102:12202 --log-opt gelf-compression-type=none --name=accountservice --replicas=1 --network=my_network -p=6767:6767 kova/accountservice

docker build -t kova/vipservice vipservice/
docker service rm vipservice
docker service create --log-driver=gelf --log-opt gelf-address=udp://192.168.1.102:12202 --log-opt gelf-compression-type=none --name=vipservice --replicas=1 --network=my_network kova/vipservice
