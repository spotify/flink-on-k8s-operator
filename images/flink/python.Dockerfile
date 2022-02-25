ARG FLINK_VERSION
ARG SCALA_VERSION
FROM flink:${FLINK_VERSION}-scala_${SCALA_VERSION}
ARG FLINK_HADOOP_VERSION
ARG GCS_CONNECTOR_VERSION
ARG FLINK_VERSION
ARG PYTHON_VERSION

RUN test -n "$FLINK_VERSION"
RUN test -n "$FLINK_HADOOP_VERSION"
RUN test -n "$GCS_CONNECTOR_VERSION"

ARG GCS_CONNECTOR_NAME=gcs-connector-${GCS_CONNECTOR_VERSION}.jar
ARG GCS_CONNECTOR_URI=https://storage.googleapis.com/hadoop-lib/gcs/${GCS_CONNECTOR_NAME}
ARG FLINK_HADOOP_JAR_NAME=flink-shaded-hadoop-2-uber-${FLINK_HADOOP_VERSION}.jar
ARG FLINK_HADOOP_JAR_URI=https://repo.maven.apache.org/maven2/org/apache/flink/flink-shaded-hadoop-2-uber/${FLINK_HADOOP_VERSION}/${FLINK_HADOOP_JAR_NAME}

# Download and configure GCS connector.
# When running on GKE, there is no need to enable and include service account
# key file, GCS connector can get credential from VM metadata server.
RUN echo "Downloading ${GCS_CONNECTOR_URI}" && \
  wget -q -O /opt/flink/lib/${GCS_CONNECTOR_NAME} ${GCS_CONNECTOR_URI}
RUN echo "Downloading ${FLINK_HADOOP_JAR_URI}" && \
  wget -q -O /opt/flink/lib/${FLINK_HADOOP_JAR_NAME} ${FLINK_HADOOP_JAR_URI}

# Install Python and pyflink .
RUN apt-get update && \
  apt-get install -y build-essential libssl-dev zlib1g-dev libbz2-dev libffi-dev && \
  wget https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tgz && \
  tar -xvf Python-${PYTHON_VERSION}.tgz && \
  cd Python-${PYTHON_VERSION} && \
  ./configure --without-tests --enable-shared && \
  make -j6 && \
  make install && \
  ldconfig /usr/local/lib && \
  cd .. && rm -f Python-${PYTHON_VERSION}.tgz && rm -rf Python-${PYTHON_VERSION} && \
  ln -s /usr/local/bin/python3 /usr/local/bin/python && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* && \
  pip3 install apache-flink==${FLINK_VERSION}
