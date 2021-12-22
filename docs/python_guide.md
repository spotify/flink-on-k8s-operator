# Running Python jobs using pyflink API with the Flink Operator

This guide provides how to start a job with a python application using pyflink that is unnecessary to deploy Apache Beam.

## Prerequisites

Apache Flink is not officially providing any dockerfile including python, you should deploy a dockerfile to pull it into k8s pod.

1. Build a Dockerfile: please follow [DockerSetup#enableing-python](https://nightlies.apache.org/flink/flink-docs-master/docs/deployment/resource-providers/standalone/docker/#enabling-python) in the flink docs
2. Deploy the Dockerfile to any docker registry

## Starting a job with a python file

You can start a job with a python file by specifying the `.spec.job.pythonFile` property and changing the `.spec.image.name` to yours.
The `.spec.job.pythonFile` is transformed to `python` as an argument in the flink command.

```yaml
apiVersion: flinkoperator.k8s.io/v1beta1
kind: FlinkCluster
metadata:
  name: flinkjobcluster-sample
spec:
  ...
  image:
    name: your_dockerfile
  ...
  job:
    pythonFile: "examples/python/table/word_count.py"
```

## Starting a job with one or more python files

If you wrote the application with multiple python files, speicify `.spec.job.pythonModule` and `.spec.job.pythonFiles`.
These properties are transformed to `pyModule` and `pyFiles` as arguments in the flink command, respectively.

```yaml
apiVersion: flinkoperator.k8s.io/v1beta1
kind: FlinkCluster
metadata:
  name: flinkjobcluster-sample
spec:
  ...
  image:
    name: your_dockerfile
  ...
  job:
    pythonModule: "word_count"
    pythonFiles: "examples/python/table"
```
