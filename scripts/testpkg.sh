#!/usr/bin/env zsh
cd pkg
go test -v -tags local -gcflags "all=-trimpath=/home/loki/work/loki/indra-labs/indra" ./...