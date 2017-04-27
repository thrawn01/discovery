## Cluster agnostic service discovery

```golang
// Discover etcd targets in cluster with ports named 'client' for 'tcp'
targets, err := discovery.Services("etcd", "client", "tcp")
if err != nil {
    return err
}

// Outputs: `Target: etcd-0.default.cluster.local`
fmt.Printf("Target: %s\n", targets[0].Target)

endpoints, err := discovery.Format(targets, "http://{{.Target}}:{{.Port}}")
if err != nil {
    return err
}

// Outputs: `http://etcd-0.default.cluster.local:2379`
fmt.Printf("Endpoints: %s\n", endpoints[0])
```