[![Build Status](https://github.com/spotify/flink-on-k8s-operator/actions/workflows/ci.yml/badge.svg)](https://github.com/spotify/flink-on-k8s-operator/actions/workflows/ci.yml)
[![GoDoc](https://pkg.go.dev/badge/github.com/spotify/flink-on-k8s-operator)](https://pkg.go.dev/github.com/spotify/flink-on-k8s-operator)
[![License](https://img.shields.io/badge/LICENSE-Apache2.0-ff69b4.svg)](http://www.apache.org/licenses/LICENSE-2.0.html)
[![Go Report Card](https://goreportcard.com/badge/github.com/spotify/flink-on-k8s-operator)](https://goreportcard.com/report/github.com/spotify/flink-on-k8s-operator)

# Kubernetes Operator for Apache Flink

[Kubernetes](https://kubernetes.io/) operator for that acts as control plane to manage the complete deployment lifecycle of [Apache Flink](https://flink.apache.org/) applications. This is an open source fork of [GoogleCloudPlatform/flink-on-k8s-operator](https://github.com/GoogleCloudPlatform/flink-on-k8s-operator) with several new features and bug fixes.

## Project Status

_Beta_

The operator is under active development, backward compatibility of the APIs is not guaranteed for beta releases.

## Prerequisites

- Version >= 1.16 of [Kubernetes](https://kubernetes.io)
- Version >= 1.7 of [Apache Flink](https://flink.apache.org)
- Version >= 1.5.3 of [cert-manager](https://cert-manager.io)

## Overview

The Kubernetes Operator for Apache Flink extends the vocabulary (e.g., Pod, Service, etc) of the Kubernetes language
with custom resource definition [FlinkCluster](docs/crd.md) and runs a
[controller](controllers/flinkcluster_controller.go) Pod to keep watching the custom resources.
Once a FlinkCluster custom resource is created and detected by the controller, the controller creates the underlying
Kubernetes resources (e.g., JobManager Pod) based on the spec of the custom resource. With the operator installed in a
cluster, users can then talk to the cluster through the Kubernetes API and Flink custom resources to manage their Flink
clusters and jobs.

## Features

- Support for both Flink [job cluster](config/samples/flinkoperator_v1beta1_flinkjobcluster.yaml) and
  [session cluster](config/samples/flinkoperator_v1beta1_flinksessioncluster.yaml) depending on whether a job spec is
  provided
- Custom Flink images
- Flink and Hadoop configs and container environment variables
- Init containers and sidecar containers
- Remote job jar
- Configurable namespace to run the operator in
- Configurable namespace to watch custom resources in
- Configurable access scope for JobManager service
- Taking savepoints periodically
- Taking savepoints on demand
- Restarting failed job from the latest savepoint automatically
- Cancelling job with savepoint
- Cleanup policy on job success and failure
- Updating cluster or job
- Batch scheduling for JobManager and TaskManager Pods
- GCP integration (service account, GCS connector, networking)
- Support for Beam Python jobs

## Installation

The operator is still under active development, there is no Helm chart available yet. You can follow either

- [User Guide](docs/user_guide.md) to deploy a released operator image on `ghcr.io/spotify/flink-operator` to your Kubernetes
  cluster or
- [Developer Guide](docs/developer_guide.md) to build an operator image first then deploy it to the cluster.

## Documentation

### Quickstart guides

- [User Guide](docs/user_guide.md)
- [Developer Guide](docs/developer_guide.md)

### API

- [Custom Resource Definition (v1beta1)](docs/crd.md)

### How to

- [Manage savepoints](docs/savepoints_guide.md)
- [Use remote job jars](config/samples/flinkoperator_v1beta1_remotejobjar.yaml)
- [Run Apache Beam Python jobs](docs/beam_guide.md)
- [Use GCS connector](images/flink/README.md)
- [Test with Apache Kafka](docs/kafka_test_guide.md)
- [Create Flink job clusters with Helm Chart](docs/flink_job_cluster_guide.md)
- [Run Python job using pyflink API](docs/python_guide.md)

### Tech talks

- CNCF Webinar: Apache Flink on Kubernetes Operator ([video](https://www.youtube.com/watch?v=MXj4lo8XHUE), [slides](docs/apache-flink-on-kubernetes-operator-20200212.pdf))

## Community

- Check out [who is using the Kubernetes Operator for Apache Flink](docs/who_is_using.md).

## Contributing

Please check [CONTRIBUTING.md](CONTRIBUTING.md) and the [Developer Guide](docs/developer_guide.md) out.
