## Setup
The following was deployed within a Minikube VM with 4 CPUs and 8GB RAM specs

1. [Namespace](#namespace)
2. [Prometheus Operator](#prometheus-operator)
3. [MinIO](#minio)
4. [Thanos](#thanos)
5. [Decoding SDK](#decoding-sdk)
6. [Log Parser](#log-parser)
7. [Loki](#loki)
8. [Grafana](#grafana)

### Namespace <a id="namespace"></a>
Create `monitoring`, `minio` and `decoding-sdk` namespaces:
```zsh
kubectl apply -f namespace.yaml
```

### Prometheus Operator <a id="prometheus-operator"></a>

Switch to `monitoring` namespace:
```zsh
kubectl config set-context --current --namespace=monitoring
```

Create Prometheus Operator Custom Resource Definitions (CRDs):
```zsh
kubectl create -f prometheus-operator-crds
```

Apply Prometheus Operator folder:
```zsh
kubectl apply -R -f prometheus-operator
```

When Prometheus Operator is up, apply Prometheus folder:
```zsh
kubectl apply -f prometheus
```

#### Checkpoint 1 <a id="checkpoint1"></a>

Port forward Prometheus Operator service:
```zsh
kubectl port-forward svc/prometheus-operated 9090
```
Visit `localhost:9090` and navigate to `Status` > `Targets`

You should see 1 active Service Monitor target with 2 endpoints as shown below:

![image](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/517ba104-b4e0-4cdb-bdd4-abb75422a32d)

*Note: We see two endpoints here because the `prometheus/prometheus.yaml` file currently specifies 2 replicas

If the targets cannot be seen, please redo the Prometheus Operator steps and make sure that K8s objects created when applying the `prometheus-operator` folder are up before applying the `prometheus` folder

### MinIO <a id="minio"></a>

Switch to `minio` namespace:
```zsh
kubectl config set-context --current --namespace=minio
```

Apply MinIO folder:
```zsh
kubectl apply -f minio
```

Visit MinIO on `cluster-ip:30001` and login with username `minioadmin` and password `minioadmin`

*Note: username and passsword is configured in `minio/secrets.yaml`

Go to `Access Key` > `Create access key` and click on `Create`

![Minio Access Key ss](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/aa99f105-0a55-48ce-9b86-617edfe07a77)

A one-time popup showing the `Access Key` and `Secret Key` will appear. Please copy the keys somewhere for now

Go to `Buckets`, enter the Bucket Name as `prometheus-metrics` and click on `Create Bucket`:

![Minio Create Bucket](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/44c7cedb-74ca-47f1-9c00-6389a07e48f3)

Update the placeholders in `prometheus/objectstore.yaml` with your `Access Key` and `Secret Key`

Apply the changes:
```zsh
kubectl apply -f prometheus/objectstore.yaml
```

### Checkpoint 2 <a id="checkpoint2"></a>

Visit Prometheus Operator on `localhost:9090` and there should be a new MinIO Service Monitor target added as shown below:

![image](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/e4bf52fe-b8d0-43cf-930f-227c4b484409)

### Thanos <a id="thanos"></a>

Switch to `monitoring` namespace:
```zsh
kubectl config set-context --current --namespace=monitoring
```

Apply Thanos folder:

```zsh
kubectl apply -f thanos
```

Run `kubectl get all` and wait till the all the objects are up. The `monitoring` namespace should look something similar to this:

![image](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/50a15f07-1af4-47cb-966b-b6f696671601)

*Note: If the storegateway pod is failing to start, make sure the bucket name in `prometheus/objectstore.yaml` matches the bucket name that was created in MinIO. Also make sure that the access and secret keys match the ones you created earlier. If you forgot to save the access and secret keys earlier, just create a new pair and update `prometheus/objectstore.yaml` accordingly

Apply Thanos Receiver folder:

```zsh
kubectl apply -f receiver
```

#### Checkpoint 3

Port forward Thanos Querier service:
```
kubectl port-forward svc/querier 9090
OR
# If your Prometheus Operator is still running on 9090, port forward to a separate port instead
kubectl port-forward svc/querier 9091:9090
```

Visit `localhost:9090` and navigate to `Stores`

You should see 1 Thanos Receiver and 1 Thanos Store Gateway that are both up:

![image](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/080b8807-9c18-4da9-bbb5-5a0934722b22)

Navigate to `Graph` and try to query for `prometheus_http_requests_total` metric:

![image](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/f5848ab8-329a-44aa-9c30-a9660d8ee80f)

If this works it means that the Thanos Querier is successfully retrieving metrics from the Thanos Receiver! 

*Note: Thanos Receiver by default uploads to the MinIO bucket every 2 hours. To test whether the Thanos Querier is successfully retrieving metrics from the Thanos Store Gateway, just observe whether your are able to query for metrics older than 2 hours

### Decoding SDK <a id="decoding-sdk"></a>

If you have already deployed the decoding sdk server and worker, you can skip this step

Else, you can deploy them by running:
```zsh
kubectl apply -f decoding-sdk
```
The server and worker will be deployed in the `decoding-sdk` namespace

If you are on Minikube, you will need to mount the models folder locally by running:
```zsh
# Replace your-models-folder-path accordingly
minikube mount your-models-folder-path:/opt/models
```

*Note: The `decoding-sdk/pv.yaml` file `hostPath` path value is set to `/opt/models`. If the models are in a separate directory, please change the path value accordingly.

### Log Parser <a id="log-parser"></a>

The Log Parser scrapes logs from the decoding sdk server and worker pods. It implements custom logic to parse the logs and export custom prometheus metrics

Since the log parser is deployed in the `monitoring` namespace and needs to scrape pod logs in the `decoding-sdk` namespace, it needs a service account with the necessary cluster role permissions

Apply the Log Parser folder:
```zsh
kubectl apply -R -f log-parser
```

#### Checkpoint 4

To test if the Log Parser is exporting metrics successfully, we need to send some dummy requests to the decoding sdk server

Create and activate virtual Python env (optional but recommended):
```zsh
python -m venv venv
source venv/bin/activate
```

Install python dependencies and run audio file:
```zsh
pip install -r requirements.txt
# Replace cluster-ip with your K8s cluster's ip address
python client_sdk_v2.py -u ws://cluster-ip:30080/abx/ws/speech -m Abax_English_ASR_0822 audio-files/countries.wav
```
If u see the text `countries` appended to test.log file, it means that the request was successful

Port forward Log Parser service:
```zsh
kubectl port-forward svc/log-parser-service 8080
```

Visit the Log Parser on `localhost:8080`, scroll down and you should see the metrics being populated with values:

![image](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/8d0611a9-d5c6-4788-89c9-19e793b02160)

### Loki <a id="loki"></a>

To setup Loki, we will leverage on the `grafana/loki-stack` helm chart to simplify the deployment process. We will be supplying the helm chart with a custom `values.yaml` file which will only enable Loki and Promtail

Add Grafana helm repo:
```zsh
helm repo add grafana https://grafana.github.io/helm-charts
```

Install grafana/loki-stack helm chart with custom values:
```zsh
helm install --namespace=monitoring --values loki/values.yaml loki grafana/loki-stack
```

We can create custom labels from the logs that Promtail scrapes, in this case we want the `status` from the response object as a custom label

To do that we need to delete the existing Promtail config secret and provide our custom Promtail config secret:
```zsh
kubectl delete secrets loki-promtail
kubectl create secret generic loki-promtail --from-file=./loki/promtail.yaml
```

Reload Promtail by deleting the Promtail pod:
```zsh
# Your pod name suffix would probably be different
kubectl delete pod/loki-promtail-brb97
```

#### Checkpoint 5
Ensure that both the Loki & Promtail pods are up and running:

![Loki Promtail Pod](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/0fc95f5b-728f-449f-a73a-45866812f8c9)

### Grafana <a id="grafana"></a>

Create Grafana folder needed by Persistent Volume:
```zsh
# Cloud
mkdir /tmp/grafana-pv

# Minikube
# Replace your-grafana-folder-path accordingly
minikube mount your-grafana-folder-path:/tmp/grafana-pv
```
*Note: If you want to use a different path, update hostPath path value in `grafana/pv.yaml`

Apply Grafana folder:
```zsh
kubectl apply -f grafana
```

Port forward Grafana service:
```zsh
kubectl port-forward svc/grafana 3000
```

Visit Grafana on `localhost:3000` and login with username `admin` and password `admin`

Adding Thanos Querier data source:
- Name: Thanos Querier
- Prometheus server URL: http://querier.monitoring.svc.cluster.local:9090

[Screencast from 2024-04-28 11-40-52.webm](https://github.com/Niflnir/k8s-cloud-fyp/assets/70419463/00b9a329-7ffc-46bd-a721-60c50dfea2ec)

Adding Loki data source:
- Name: Loki
- URL: http://loki:3100
