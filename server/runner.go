package main

import (
	"context"
	"github.com/arthurfabre/scheduler/api"
	"github.com/coreos/etcd/clientv3"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	_ "github.com/opencontainers/runc/libcontainer/nsenter"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

// Allow us to use ourselves as the container init
// nicked from https://github.com/opencontainers/runc/tree/master/libcontainer#using-libcontainer
func init() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runtime.GOMAXPROCS(1)
		runtime.LockOSThread()
		factory, _ := libcontainer.New("")
		if err := factory.StartInitialization(); err != nil {
			log.Fatalln(err)
		}
		log.Fatalln("libcontainer init error, factory.StartInitialization() returned")
	}
}

var defaultMountFlags = unix.MS_NOEXEC | unix.MS_NOSUID | unix.MS_NODEV

func config(rootFs string, cgroupName string) *configs.Config {
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

	// nicked from https://github.com/opencontainers/runc/tree/master/libcontainer#using-libcontainer
	return &configs.Config{
		Rootfs: rootFs,
		Capabilities: &configs.Capabilities{
			Bounding: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_FOWNER",
				"CAP_MKNOD",
				"CAP_NET_RAW",
				"CAP_SETGID",
				"CAP_SETUID",
				"CAP_SETFCAP",
				"CAP_SETPCAP",
				"CAP_NET_BIND_SERVICE",
				"CAP_SYS_CHROOT",
				"CAP_KILL",
				"CAP_AUDIT_WRITE",
			},
			Effective: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_FOWNER",
				"CAP_MKNOD",
				"CAP_NET_RAW",
				"CAP_SETGID",
				"CAP_SETUID",
				"CAP_SETFCAP",
				"CAP_SETPCAP",
				"CAP_NET_BIND_SERVICE",
				"CAP_SYS_CHROOT",
				"CAP_KILL",
				"CAP_AUDIT_WRITE",
			},
			Inheritable: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_FOWNER",
				"CAP_MKNOD",
				"CAP_NET_RAW",
				"CAP_SETGID",
				"CAP_SETUID",
				"CAP_SETFCAP",
				"CAname stP_SETPCAP",
				"CAP_NET_BIND_SERVICE",
				"CAP_SYS_CHROOT",
				"CAP_KILL",
				"CAP_AUDIT_WRITE",
			},
			Permitted: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_FOWNER",
				"CAP_MKNOD",
				"CAP_NET_RAW",
				"CAP_SETGID",
				"CAP_SETUID",
				"CAP_SETFCAP",
				"CAP_SETPCAP",
				"CAP_NET_BIND_SERVICE",
				"CAP_SYS_CHROOT",
				"CAP_KILL",
				"CAP_AUDIT_WRITE",
			},
			Ambient: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_FOWNER",
				"CAP_MKNOD",
				"CAP_NET_RAW",
				"CAP_SETGID",
				"CAP_SETUID",
				"CAP_SETFCAP",
				"CAP_SETPCAP",
				"CAP_NET_BIND_SERVICE",
				"CAP_SYS_CHROOT",
				"CAP_KILL",
				"CAP_AUDIT_WRITE",
			},
		},
		Namespaces: configs.Namespaces([]configs.Namespace{
			{Type: configs.NEWNS},
			{Type: configs.NEWUTS},
			{Type: configs.NEWIPC},
			{Type: configs.NEWPID},
			{Type: configs.NEWUSER},
			{Type: configs.NEWNET},
		}),
		Cgroups: &configs.Cgroup{
			Name:   cgroupName,
			Parent: "system",
			Resources: &configs.Resources{
				MemorySwappiness: nil,
				AllowAllDevices:  nil,
				AllowedDevices:   configs.DefaultAllowedDevices,
			},
		},
		MaskPaths: []string{
			"/proc/kcore",
			"/sys/firmware",
		},
		ReadonlyPaths: []string{
			"/proc/sys", "/proc/sysrq-trigger", "/proc/irq", "/proc/bus",
		},
		Devices:  configs.DefaultAutoCreatedDevices,
		Hostname: "testing",
		Mounts: []*configs.Mount{
			{
				Source:      "proc",
				Destination: "/proc",
				Device:      "proc",
				Flags:       defaultMountFlags,
			},
			{
				Source:      "tmpfs",
				Destination: "/dev",
				Device:      "tmpfs",
				Flags:       unix.MS_NOSUID | unix.MS_STRICTATIME,
				Data:        "mode=755",
			},
			{
				Source:      "devpts",
				Destination: "/dev/pts",
				Device:      "devpts",
				Flags:       unix.MS_NOSUID | unix.MS_NOEXEC,
				Data:        "newinstance,ptmxmode=0666,mode=0620,gid=5",
			},
			{
				Device:      "tmpfs",
				Source:      "shm",
				Destination: "/dev/shm",
				Data:        "mode=1777,size=65536k",
				Flags:       defaultMountFlags,
			},
			{
				Source:      "mqueue",
				Destination: "/dev/mqueue",
				Device:      "mqueue",
				Flags:       defaultMountFlags,
			},
			{
				Source:      "sysfs",
				Destination: "/sys",
				Device:      "sysfs",
				Flags:       defaultMountFlags | unix.MS_RDONLY,
			},
		},
		UidMappings: []configs.IDMap{
			{
				ContainerID: 0,
				HostID:      1000,
				Size:        65536,
			},
		},
		GidMappings: []configs.IDMap{
			{
				ContainerID: 0,
				HostID:      1000,
				Size:        65536,
			},
		},
		Networks: []*configs.Network{
			{
				Type:    "loopback",
				Address: "127.0.0.1/0",
				Gateway: "localhost",
			},
		},
		Rlimits: []configs.Rlimit{
			{
				Type: unix.RLIMIT_NOFILE,
				Hard: uint64(1025),
				Soft: uint64(1025),
			},
		},
	}
}

type Runner struct {
	client *clientv3.Client
	id     *api.NodeID
}

func (r *Runner) Start(containerDir string, rootFs string) {
	rootFs, err := filepath.Abs(rootFs)
	if err != nil {
		log.Fatalln("Error getting absolute rootfs path", err)
	}

	factory, err := libcontainer.New(containerDir, libcontainer.Cgroupfs, libcontainer.InitArgs(os.Args[0], "init"))
	if err != nil {
		log.Fatalln("Error creating libcontainer factory:", err)
	}

	// TODO - What does cgroupName do?
	cfg := config(rootFs, "test")

	newTasks := watchQueuedTasks(context.Background(), r.client)

	for task := range newTasks {
		if err := task.run(context.Background(), r.client, r.id); err != nil {
			// TODO - Differentiate stolen task from other errors
			continue
		}

		log.Println("Running task", task.Id.Uuid)

		go func(runningTask *Task) {
			container, err := factory.Create(task.Id.Uuid, cfg)
			if err != nil {
				log.Println("Error creating container", err)
				return
			}

			log.Println(container.Processes())
		}(task)
	}
}
