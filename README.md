* Kubernetes like thing

# Usage

## Requirements

* `make`

* `protoc`

* `protoc-gen-go`: https://github.com/golang/protobuf/

* `go dep`


## Build

* `dep ensure`: Download & vendor all dependencies

* `make`: Build `./scheduler`


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
    * Etcd leader monitors this keyspace for DELETES - indicate a node has gone
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
    * Could be replaced with a proxy in some (most) nodes
        * No design limits preventing this implementation

* Single node (Etcd leader) in charge of cleanup
    * Distributed stealing algorithm could be implemented instead

* All comms are over HTTP, not HTTPS


# TODO

* Remove old finished jobs from etcd

* Testing

* Database compaction & defrag


# Background docs

* etcd API: https://github.com/coreos/etcd/blob/master/Documentation/learning/api.md

* gRPC error handling: https://github.com/avinassh/grpc-errors/tree/master/go
