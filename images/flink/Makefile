include properties

all: build

# Build the docker image
build:
	cd docker
ifeq ($(PYTHON_VERSION),)
	docker build . \
		-t ${IMAGE_TAG} \
		--build-arg FLINK_VERSION=${FLINK_VERSION} \
		--build-arg SCALA_VERSION=${SCALA_VERSION} \
		--build-arg FLINK_HADOOP_VERSION=${FLINK_HADOOP_VERSION} \
		--build-arg GCS_CONNECTOR_VERSION=${GCS_CONNECTOR_VERSION}
else
	docker build . \
		-f Dockerfile.python \
		-t ${IMAGE_TAG} \
		--build-arg FLINK_VERSION=${FLINK_VERSION} \
		--build-arg SCALA_VERSION=${SCALA_VERSION} \
		--build-arg FLINK_HADOOP_VERSION=${FLINK_HADOOP_VERSION} \
		--build-arg GCS_CONNECTOR_VERSION=${GCS_CONNECTOR_VERSION} \
		--build-arg PYTHON_VERSION=${PYTHON_VERSION}
endif

# Push the docker image
push:
	docker push ${IMAGE_TAG}
