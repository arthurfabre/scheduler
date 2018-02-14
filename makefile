# List of source files
PROTOS:=schedapi/api.pb.go schedserver/task.pb.go
SRC:=main.go $(wildcard schedserver/*.go)

# Commands
PROTOC:=protoc
GO_PROTOC_FLAGS:=
GO:=go

# Single comma
COMMA:=,

# Build server
scheduler: $(PROTOS) $(SRC)
	$(GO) build

# Override go package name to import api & depend on it
schedserver/task.pb.go: schedapi/api.proto
schedserver/task.pb.go: GO_PROTOC_FLAGS+= Mschedapi/api.proto=github.com/arthurfabre/scheduler/schedapi

# Compile proto definition
%.pb.go: %.proto
	$(PROTOC) $*.proto --go_out=plugins=grpc$(foreach f,$(GO_PROTOC_FLAGS),$(COMMA)$f):./
