# Cost Reports
<a href='https://github.com/jpoles1/gopherbadger' target='_blank'>![gopherbadger-tag-do-not-edit](https://img.shields.io/badge/Go%20Coverage-76%25-brightgreen.svg?longCache=true&style=flat)</a>
[![Docker](https://github.com/empathyco/platform-cost-report/actions/workflows/docker.yml/badge.svg)](https://github.com/empathyco/platform-cost-report/actions/workflows/docker.yml)
[![Gosec](https://github.com/empathyco/platform-cost-report/actions/workflows/gosec.yaml/badge.svg)](https://github.com/empathyco/platform-cost-report/actions/workflows/gosec.yaml)
[![Reviewdog](https://github.com/empathyco/platform-cost-report/actions/workflows/reviewdog.yml/badge.svg)](https://github.com/empathyco/platform-cost-report/actions/workflows/reviewdog.yml)
## Architecture
![](docs/diagram.png)
## Requirements

```sh
brew install go
```

### IAM

To be able to query for prices you should have the following permissions:

#### AWS

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": [
                "pricing:*"
            ],
            "Effect": "Allow",
            "Resource": "*"
        },
        {
            "Action": [
                "ec2:Describe*"
            ],
            "Effect": "Allow",
            "Resource": "*"
        }
    ]
}
```
You could run the terraform code to create it.

set the following variables to be able to run the code:

- **oidc_url**: eks cluster url where you will be running the exporter.
- **role_name:** the name of the role to be created.
- **terraform_role_ARN:** the role which will create the resources.

```sh
cd terraform
terraform init
terraform apply
```

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
go run main.go
```

## Endpoints

- Metrics: localhost:8080/metrics
- Healthcheck: localhost:8080/health

## Metrics

### instance_cost_all

| Name                                   | Description       |
|----------------------------------------|-------------------|
| label_beta_kubernetes_io_instance_type | machine type      |
| label_eks_amazonaws_com_capacity_type  | instance type     |
| vcpu                                   | virtual cpu       |
| memory                                 | memory            |
| unit                                   | unit              |
| Description                            | description       |
| label_topology_kubernetes_io_zone      | availability zone |
| region                                 | region            |

### instance_cost

| Name                                   | Description       |
|----------------------------------------|-------------------|
| label_beta_kubernetes_io_instance_type | machine type      |
| label_eks_amazonaws_com_capacity_type  | instance type     |
| vcpu                                   | virtual cpu       |
| memory                                 | memory            |
| unit                                   | unit              |
| Description                            | description       |
| label_topology_kubernetes_io_zone      | availability zone |
| region                                 | region            |
### instance_mem_price

| Name                                   | Description       |
|----------------------------------------|-------------------|
| label_beta_kubernetes_io_instance_type | machine type      |
| label_eks_amazonaws_com_capacity_type  | instance type     |
| unit                                   | unit              |
| label_topology_kubernetes_io_zone      | availability zone |
| region                                 | region            |

### instance_cpu_price
| Name                                   | Description       |
|----------------------------------------|-------------------|
| label_beta_kubernetes_io_instance_type | machine type      |
| label_eks_amazonaws_com_capacity_type  | instance type     |
| unit                                   | unit              |
| label_topology_kubernetes_io_zone      | availability zone |
| region                                 | region            |

### instance_capacity

| Name                                   | Description       |
|----------------------------------------|-------------------|
| label_beta_kubernetes_io_instance_type | machine type      |
| label_eks_amazonaws_com_capacity_type  | instance type     |
| unit                                   | unit              |
| label_topology_kubernetes_io_zone      | availability zone |
| region                                 | region            |

### instance_discount

| Name                                   | Description       |
|----------------------------------------|-------------------|
| label_beta_kubernetes_io_instance_type | machine type      |
| label_eks_amazonaws_com_capacity_type  | instance type     |
| unit                                   | unit              |
| label_topology_kubernetes_io_zone      | availability zone |
| region                                 | region            |