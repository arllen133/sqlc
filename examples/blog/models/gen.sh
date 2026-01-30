#!/bin/bash
# 代码生成脚本

echo "Generating ORM code for blog models..."
go run ../../../cmd/sqlcli/main.go \
  -model . \
  -output . \
  -module github.com/arllen133/sqlc/examples/blog \
  -package models
echo "Done!"


