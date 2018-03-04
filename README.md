* Kubernetes like thing

# Requirements

* `make`

* `protoc`

* [`protoc-gen-go`](https://github.com/golang/protobuf/)
    * `go get -u github.com/golang/protobuf/protoc-gen-go`

* [`dep`](https://github.com/golang/dep)
    * `go get -u github.com/golang/dep/cmd/dep`

* `$GOPATH/bin` in `$PATH` if `go get` is used to install `dep` or `protoc-gen-go`

* Suitable rootfs for running a container
    * With `docker`: `docker export $(docker create busybox) | tar -C rootfs/ -xvf -`

# Build

* `make`: Build `client` and `server`

# Usage

* See `./server.elf --help` and `./client.elf --help`

* Two node cluster:
`sudo ./server.elf --data-dir (mktemp -d) 127.0.0.1 node1 -N "node2=http://127.0.0.2:2380" -n -r rootfs/`
`sudo ./server.elf --data-dir (mktemp -d) 127.0.0.2 node2 -N "node1=http://127.0.0.1:2380" -n -r rootfs/`

* Submit task:
`./client.elf -N 127.0.0.2:8080 run ls -- -l`

# Design

* Fully distributed (ie no distinction between scheduler / worker). Every node has:
    * gRPC API for scheduling / checking jobs
    * embedded etcd datastore (https://godoc.org/github.com/coreos/etcd/embed)
        * client access / address is in 127.0.0.0/8
        * peer access / address is public
        * used for all inter-node comms
    * libcontainer https://github.com/opencontainers/runc/tree/master/libcontainer)
        * cgroups, namespaces..

* Work stealing algorithm
    * Nodes can watch "queued" namespace in etcd - see the [key schema section](#etcd-key-schema)
    * Use transaction to do this atomically
    * Pros:
        * Available resources don't have to be propagated throughout the cluster
    * Cons:
        * Might be high contention on "stealing" tasks
            * Could be solved with small delay between watch event and stealing
        * Work distribution might not be very fair
            * Delay solution above could be proportional to node loading (thanks Kevin!)

* Nodes watch tasks they're running to see if they've been stopped / canceled

* Nodes generate unique UUID for themselves, and store in etcd with a lease
    * All nodes monitor this keyspace for DELETES - indicate a node has gone
        * Its tasks are sent back to "queued"


## ETCD Key Schema

* One prefix for jobs, with proto Task values
    * `/task/UUID -> Task Proto`
* Separate prefixes for:
    * queued
        * `/task/status/queued/UUID -> NULL`
    * running
        * `/task/status/running/NODE_ID/UUID -> NULL`
            * `NODE_ID` is the UUID of the node running the task
    * complete
        * `/task/status/complete/EPOCH/UUID -> NULL`
            * `EPOCH` is the UNIX Epoch at which the task was completed
    * canceled
        * `/task/status/canceled/EPOCH/UUID -> NULL`
            * `EPOCH` is the UNIX Epoch at which the task was canceled
    * only keys, no values (doesn't seem supported, might have to use empty string)

* Pros:
    * Allows simple O(1) job retrieval
    * Allows easy watching of queued jobs
    * Easy to move all tasks of failed node to complete or pending (if we retry tasks)
    * Easy to remove old completed / cancelled t
* Cons:
    * Lots of keys
    * Need to keep proto Task value status in sync with prefixes tracking Task status


## API

* Messages:
    * Task
        * ID (UUID?)
        * Status
            * Pending
            * Running
                * Node id?
            * Complete
                * Exit code?
        * Command
        * Arguments
        * Requirements
            * RAM
            * CPU
            * Disk IO?
        * Restart on cluster error (ie node crashes)

* sumbit()
    * Add task to etcd "pending"
    * Return UUID

* cancel()

* status()

* log()


# Limitations

* Logs are not distributed, stored on single node
    * Shouldn't store big things in etcd

* Work distribution could be unfair (see Work Stealing Algorithm)

* Etcd is embedded in every node, limiting max cluster size (etc recommends 7 max)
    * Could be turned off in some (most) nodes
        * No design limits preventing this implementation

* Stealing algorithm is used to deal with node failures
    * Can be expensive

* All comms are over HTTP, not HTTPS

* Server must be run as `root`
    * Can't configure `cgroups` otherwise


# TODO

* Remove old finished tasks
    * Can be listed using `listDoneTasks()` from `task.go`
    * Need to implement `Task.delete()`

* Resource constraints / limits
    * Need to be added to `api.proto` in `TaskRequest`
    * Use [procfs](https://godoc.org/github.com/prometheus/procfs) to monitor system usage
    * Check task doesn't exceed system usage before stealing it in `runner.go`

* Failed node task migration
    * Nodes need to obtain a lease, publish their UUID in `/node`, and watch `/node` for `DELETE`s
    * On `DELETE`, every node tries to:
        * list all the running tasks of the failed node using `listNodeTasks()` from `task.go`
        * Requeue every task found

## Tests

### ETCD

* Startup cancellation
* Successful startup
* etcd client can connect

### Task

* Fetching and updating
    * status keys remain in sync
* updating out of date task results in ConcurrentTaskModErr
* `taskID()` parsing of keys works
* error when unmarshaling invalid task
* error when unmarshaling task with mis-matched ID
* `watch()` doesn't leak ressources on error or ctx.Cancel()

### Runner

* container creation and running
* `watchCancel()` returns when task is canceled
* `watchCancel()` returns when ctx.Cancel()
* non zero process exit codes don't cause an error


# Background docs

* etcd API: https://github.com/coreos/etcd/blob/master/Documentation/learning/api.md

* gRPC error handling: https://github.com/avinassh/grpc-errors/tree/master/go
