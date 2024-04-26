## Setup
The following instructions below are to be executed in order

### Namespace
Create `monitoring` and `minio` namespaces
```zsh
kubectl apply -f namespace.yaml
```

### Prometheus Operator
Switch to `monitoring` namespace
```zsh
kubectl config set-context --current --namespace=monitoring
```

Create Prometheus Operator Custom Resource Definitions (CRDs)
```zsh
kubectl create -f prometheus-operator-crds
```

Apply Prometheus Operator folder
```zsh
kubectl apply -R -f prometheus-operator
```

When Prometheus Operator is up, apply Prometheus folder
```zsh
kubectl apply -f prometheus
```

### Checkpoint 1
Port forward Prometheus Operator service
```zsh
kubectl port-forward svc/prometheus-operated 9090
```
Visit `localhost:9090` and navigate to `Status` > `Targets`\

You should see 2 active targets as seen below

*Note: We see two targets here because the `prometheus/prometheus.yaml` file currently specifies 2 replicas

![image](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/517ba104-b4e0-4cdb-bdd4-abb75422a32d)

If the targets cannot be seen, please redo the Prometheus Operator steps and make sure that K8s objects created by applying the Prometheus Operator folder are up before deploying the Prometheus folder.
