# List of source files
PROTOS:=schedapi/api.pb.go schedserver/task.pb.go
SRC:=main.go $(wildcard schedserver/*.go)

# Commands
PROTOC:=protoc
GO_PROTOC_FLAGS:=
GO:=go

# Single comma
COMMA:=,

# Include proto dependency 
PROTO_DEPS:=$(PROTOS:.pb.go=.d)
-include $(PROTO_DEPS)

# Build server
.DEFAULT_GOAL:=scheduler
scheduler: $(PROTOS) $(SRC)
	$(GO) build

# Override go package name to depend on it
# TODO - Could we do this automatically from depdency info somehow?
schedserver/task.pb.go: GO_PROTOC_FLAGS+= Mschedapi/api.proto=github.com/arthurfabre/scheduler/schedapi

# Compile proto definition
%.pb.go: %.proto
	$(PROTOC) $*.proto --dependency_out=$*.d --go_out=plugins=grpc$(foreach f,$(GO_PROTOC_FLAGS),$(COMMA)$f):./

.PHONY: clean
clean:
	rm -rf $(PROTOS) $(PROTO_DEPS)
	go clean
