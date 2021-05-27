module avenuesec/workflow-poc

go 1.12

require (
	github.com/go-redis/redis/v8 v8.9.0
	github.com/gogo/googleapis v1.3.1 // indirect
	github.com/gogo/status v1.1.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/pborman/uuid v1.2.0
	github.com/prometheus/common v0.14.0 // indirect
	github.com/prometheus/procfs v0.2.0 // indirect
	github.com/rs/cors v1.7.0
	github.com/samuel/go-thrift v0.0.0-20190219015601-e8b6b52668fe // indirect
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/uber-go/tally v3.3.11+incompatible
	go.uber.org/cadence v0.17.0
	go.uber.org/multierr v1.5.0 // indirect
	go.uber.org/net/metrics v1.1.0 // indirect
	go.uber.org/thriftrw v1.20.2 // indirect
	go.uber.org/yarpc v1.42.0
	go.uber.org/zap v1.13.0
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	google.golang.org/api v0.47.0
	google.golang.org/protobuf v1.26.0 // indirect
)

replace github.com/apache/thrift => github.com/apache/thrift v0.0.0-20190309152529-a9b748bb0e02
