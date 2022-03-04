# Running Python jobs using pyflink API with the Flink Operator

This guide demonstrates how to start a job with a python application using pyFlink without deploying Apache Beam

## Prerequisites

Apache Flink does not provide any official docker images for pyFlink, you will need to build and host your own image. A sample docker file is provided
in `images/flink/python.Dockerfile`

Alternatively:

1. Create your own Dockerfile: please follow [DockerSetup#enableing-python](https://nightlies.apache.org/flink/flink-docs-master/docs/deployment/resource-providers/standalone/docker/#enabling-python) in the flink docs
2. Deploy the Dockerfile to any docker registry

## Starting a job with a python file

You can start a job with a python file by specifying the `.spec.job.pyFile` property. The `.spec.job.pyFile` is transformed to `python` 
as an argument in the flink command.

Make sure you update `.spec.image.name` to point to your pyFlink Docker Image and registry.

```yaml
apiVersion: flinkoperator.k8s.io/v1beta1
kind: FlinkCluster
metadata:
  name: flinkjobcluster-sample
spec:
  ...
  image:
    name: <your_dockerfile>
  ...
  job:
    pyFile: "examples/python/table/word_count.py"
```

## Starting a job with one or more python files

If you wrote the application with multiple python files, specify `.spec.job.pyModule` and `.spec.job.pyFiles`.
These properties are transformed to `pyModule` and `pyFiles` as arguments in the flink command, respectively.
Refer to the [pyFlink CLI Docs](https://nightlies.apache.org/flink/flink-docs-release-1.14/docs/deployment/cli/#submitting-pyflink-jobs) for further
information.

```yaml
apiVersion: flinkoperator.k8s.io/v1beta1
kind: FlinkCluster
metadata:
  name: flinkjobcluster-sample
spec:
  ...
  image:
    name: <your_dockerfile>
  ...
  job:
    pyModule: "word_count"
    pyFiles: "examples/python/table"
```
