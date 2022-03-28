# Nats test

Run `go run main.go`

```
  -debug
    	Print debug info
  -islands int
    	Total islands to distribute peers between (default 10)
  -peers int
    	Total peers to simulate (default 10000)
  -url string
    	Nats cluster URL (default "nats://127.0.0.1:4222")
```

# Our test

Server was:

```
Image
 Ubuntu 20.04 (LTS) x64
Size
4 vCPUs
8GB / 160GB Disk
San Francisco
```

We had two clients running 6 supernodes each, 500 peers every supernode
We had an extra client droplet running 4 supernodes, 500 peers each
