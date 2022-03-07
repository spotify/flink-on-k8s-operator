ARG FLINK_VERSION
ARG SCALA_VERSION
FROM flink:${FLINK_VERSION}-scala_${SCALA_VERSION}
ARG FLINK_HADOOP_VERSION
ARG GCS_CONNECTOR_VERSION

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
