#!/bin/sh

SOURCE_DIR="$(dirname $(readlink -f $0))"
cd "$SOURCE_DIR"

if [ -z ${1+x} ] || [ -z ${2+x} ]; then
	echo "Usage: ./deploy SERVER_LOGIN PEM_PATH"
	exit
fi

rm -f scout.tar.gz # clean up old deploys if there was a problem

# build the deployed archive
go install scout
tar -czf scout.tar.gz img js css "../../bin/scout"

# put the archive on the server
printf 'put scout.tar.gz' | sftp  -i "$2" "$1"
printf 'rm -rf scout; mkdir scout; cd scout; tar -xzf ../scout.tar.gz; rm ../scout.tar.gz' | ssh  -i "$2" "$1"

rm scout.tar.gz # clean up