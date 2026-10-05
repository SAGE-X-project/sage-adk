# Protobuf bindings

Generated Go bindings for the existing `agent.proto` schema are committed in `pb/`
so library users can build without installing protoc. They were generated with
protoc 6.31.1 (grpcio-tools 1.76.0), protoc-gen-go 1.36.11 and
protoc-gen-go-grpc 1.5.1. `generate.sh` accepts a standard protoc installation and
prints pinned plugin installation commands if the plugins are missing.

No schema change is part of this migration. The existing gRPC implementation is a
legacy plaintext transport and has not been assessed as a SAGE 0.10.0 binding.
