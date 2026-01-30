#!/bin/bash
# 代码生成脚本

echo "Generating ORM code for blog models..."
go run ../../../cmd/orm-gen/main.go \
  -model . \
  -output . \
  -module github.com/arllen133/sqlc \
  -package examples/blog/models
echo "Done!"

