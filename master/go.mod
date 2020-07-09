module github.com/tockn/takuhai/master

go 1.12

require (
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/golang/snappy v0.0.1 // indirect
	github.com/jinzhu/gorm v1.9.11
	github.com/jmoiron/sqlx v1.2.0
	github.com/lestrrat-go/tcputil v0.0.0-20180223003554-d3c7f98154fb // indirect
	github.com/lestrrat-go/test-mysqld v0.0.0-20190527004737-6c91be710371
	github.com/pkg/errors v0.8.1 // indirect
	github.com/robfig/cron/v3 v3.0.0 // indirect
	github.com/rs/xid v1.2.1
	github.com/rubenv/sql-migrate v0.0.0-20191121092708-da1cb182f00e
	github.com/shirou/gopsutil v2.19.11+incompatible
	github.com/tockn/takuhai v0.0.0-20191220054049-498e5b149794
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c // indirect
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.1.3
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/yaml.v2 v2.2.7
)

replace github.com/tockn/takuhai => ../
