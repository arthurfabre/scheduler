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
    * Nodes can watch "pending" namespace using etcd
    * Available resources don't have to be propagated 

* Nodes watch tasks they're running to see if they've been stopped / canceled

# API

* Messages:
    * Task
        * ID (UUID?)
        * Status
            * Pending
            * Running
            * Complete
                * Exit code?
        * Command
        * Arguments
        * Requirements
            * RAM
            * CPU
            * Disk IO?

* sumbit()
    * Add task to etcd "pending"
    * Return UUID

* cancel()

* status()

* log()


# Limitations

* Logs are not distributed, stored on single node
    * Shouldn't store big things in etcd


# TODO

* How does etc handle missing / disappearing nodes?
    * Need to modify status of tasks on said nodes
        * Checked by all nodes, or just leader?

* Remove old finished jobs from etcd

* Testing
