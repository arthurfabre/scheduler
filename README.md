* Kubernetes like thing


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

* Etcd leader monitors nodes (using Maintenance Alarms)
    * When a node fails, it's tasks are migrated to back to "pending" or "complete"
    * What happens if node doesn't realize it's been considered failed?
        * Does etcd handle this?
        * Task proto etcd revision will have changed, node should realize this


## ETCD Key Schema

* One prefix for jobs, with proto Task values
    * `/task/UUID -> Task Proto`
* Separate prefixes for:
    * complete
        * `/status/complete/UUID -> NULL`
    * queued
        * `/status/queued/UUID -> NULL`
    * each node (for in progress)
        * `/status/running/NODE_ID/UUID -> NULL`
    * only keys, no values (doesn't seem supported, might have to use empty string)

* Pros:
    * Allows simple O(1) job retrieval
    * Allows easy watching of queued jobs
    * Easy to move all tasks of failed node to complete or pending (if we retry tasks)
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


# TODO

* How does etc handle missing / disappearing nodes?
    * Need to modify status of tasks on said nodes
        * Checked by all nodes, or just leader?

* Remove old finished jobs from etcd

* Testing

* Database compaction?


# Background docs

* etcd API: https://github.com/coreos/etcd/blob/master/Documentation/learning/api.md

* gRPC error handling: https://github.com/avinassh/grpc-errors/tree/master/go
