# Cost Reports

## Architecture
![](docs/diagram.png)
## Requirements

```sh
brew install go
```

## Credentials

[Creditial file](credentials.json) is need it in order to query the Google Sheets.

## Build

```sh
## Binary
make build
## Docker
make docker
```
For those who wants keep it simple and avoid install a lot of things:

```sh
docker build . -t cost-report
docker run -p 8080:8080 cost-report
```

## Development

```sh
go mod vendor
go run main.go -kubeconfig $KUBECONFIG 
```

## Endpoints

- Metrics: localhost:8080/metrics
- Healthcheck: localhost:8080/health

## Metrics

### co2_node

| Name         | Description  |
|--------------|--------------|
| name         | pod name     |
| region       | region       |
| machine_type | machine type |

**Example:**
```
# HELP co2_node Cost Instance Type
# TYPE co2_node gauge
co2_node{machine_type="m4.large",name="ip-10-204-56-193.eu-west-1.compute.internal",region="eu-west-1"} 4.135008
co2_node{machine_type="m4.xlarge",name="ip-10-204-134-194.eu-west-1.compute.internal",region="eu-west-1"} 8.173808000000001
```

### watt_node

| Name         | Description  |
|--------------|--------------|
| name         | pod name     |
| region       | region       |
| machine_type | machine type |


**Example:**
```
# HELP watt_node Watt Instance Type
# TYPE watt_node gauge
watt_node{machine_type="m4.large",name="ip-10-204-56-193.eu-west-1.compute.internal",region="eu-west-1"} 7.74
watt_node{machine_type="m4.xlarge",name="ip-10-204-134-194.eu-west-1.compute.internal",region="eu-west-1"} 15.49
```

### co2_pod

| Name   | Description |
|--------|-------------|
| name   | pod name    |
| region | region      |


**Example:**
```
# HELP co2_pod Cost Pod
# TYPE co2_pod gauge
co2_pod{name="alertmanager-prometheus-kube-prometheus-alertmanager-0",region="eu-west-1"} 0.006150875929103999
co2_pod{name="applicationset-argocd-applicationset-cc7cc44d8-nwrcl",region="eu-west-1"} 0.004985003174316
```

### watt_pod
| Name   | Description |
|--------|-------------|
| name   | pod name    |
| region | region      |

**Example:**
```
# HELP watt_pod Watt Pods
# TYPE watt_pod gauge
watt_pod{name="alertmanager-prometheus-kube-prometheus-alertmanager-0",region="eu-west-1"} 0.01622066437
watt_pod{name="applicationset-argocd-applicationset-cc7cc44d8-nwrcl",region="eu-west-1"} 0.0131461054175
```


