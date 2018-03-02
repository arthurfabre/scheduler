# Commands
PROTOC:=protoc
GO_PROTOC_FLAGS:=
GO:=go
DEP:=dep

# Single comma
COMMA:=,

# List of things we need to clean
CLEAN:=

define _target
$1: DIR:=$2/
$1: $(shell find $2/ -name '*.go') $3

# Include proto dependency 
-include $(3:.pb.go=.d)

CLEAN+=$1 $(3:.pb.go=.d) $3
endef

define target
$(eval $(call _target,$1.elf,$1,$2))$1.elf
endef

CLIENT:=$(call target,client,api/api.pb.go)
SERVER:=$(call target,server,api/api.pb.go server/pb/task.pb.go)

# Override go package name to depend on it
# TODO - Could we do this automatically from depdency info somehow?
server/pb/task.pb.go: GO_PROTOC_FLAGS+= Mapi/api.proto=github.com/arthurfabre/scheduler/api

.DEFAULT_GOAL:=all
.PHONY:all
all: $(CLIENT) $(SERVER)

# Build package into executable
%.elf: vendor/
	$(GO) fmt ./$(DIR)...
	$(GO) build -o $@ ./$(DIR)

# Fetch dependencies with dep
vendor/: Gopkg.lock Gopkg.toml
	$(DEP) ensure

# Compile proto definition
%.pb.go: %.proto
	$(PROTOC) $*.proto --dependency_out=$*.d --go_out=plugins=grpc$(foreach f,$(GO_PROTOC_FLAGS),$(COMMA)$f):./

.PHONY: clean
clean:
	rm -f $(CLEAN)
