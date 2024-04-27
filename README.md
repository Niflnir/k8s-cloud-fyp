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

You should see 1 active Service Monitor target with 2 endpoints as shown below:

![image](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/517ba104-b4e0-4cdb-bdd4-abb75422a32d)

*Note: We see two endpoints here because the `prometheus/prometheus.yaml` file currently specifies 2 replicas.

If the targets cannot be seen, please redo the Prometheus Operator steps and make sure that K8s objects created when applying the `prometheus-operator` folder are up before applying the `prometheus` folder.

### MinIO
Switch to `minio` namespace
```zsh
kubectl config set-context --current --namespace=minio
```

Apply MinIO folder
```zsh
kubectl apply -f minio
```

Visit MinIO on `cluster-ip:30001` and login with username `minioadmin` and password `minioadmin`

*Note: username and passsword is configured in `minio/secrets.yaml`

Go to `Access Key` > `Create access key` and click on `Create`

![Minio Access Key ss](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/aa99f105-0a55-48ce-9b86-617edfe07a77)

A one-time popup showing the `Access Key` and `Secret Key` will appear. Please copy the keys somewhere for now.

Go to `Buckets`, enter the Bucket Name as `prometheus-metrics` and click on `Create Bucket`:

![Minio Create Bucket](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/44c7cedb-74ca-47f1-9c00-6389a07e48f3)

Update the placeholders in `prometheus/objectstore.yaml` with your `Access Key` and `Secret Key`. 

Apply the changes

```zsh
kubectl apply -f prometheus/objectstore.yaml
```

### Checkpoint 2 

Visit Prometheus Operator on `localhost:9090` and there should be a new MinIO Service Monitor target added as shown below:

![image](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/e4bf52fe-b8d0-43cf-930f-227c4b484409)
