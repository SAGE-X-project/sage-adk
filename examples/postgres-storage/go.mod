module github.com/sage-x-project/sage-adk/examples/postgres-storage

go 1.24.4

replace github.com/sage-x-project/sage-adk => ../../

require github.com/sage-x-project/sage-adk v0.0.0-00010101000000-000000000000

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/redis/go-redis/v9 v9.7.0 // indirect
)
