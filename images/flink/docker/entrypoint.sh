#!/usr/bin/env bash

# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# A wrapper around the Flink base image's entrypoint with additional setups.

set -x

echo "Flink entrypoint..."

FLINK_CONF_FILE="${FLINK_HOME}/conf/flink-conf.yaml"

download_job_file()
{
  URI=$1
  JOB_PATH=$2
  mkdir -p ${JOB_PATH}
  echo "Downloading job file ${URI} to ${JOB_PATH}"
  if [[ "${URI}" == gs://* ]]; then
    gsutil cp "${URI}" "${JOB_PATH}"
  elif [[ "${URI}" == http://* || "${URI}" == https://* ]]; then
    wget -nv -P "${JOB_PATH}" "${URI}"
  else
    echo "Unsupported protocol for ${URI}"
    exit 1
  fi
}

# Add user-provided properties to Flink config.
# FLINK_PROPERTIES is a multi-line string of "<key>: <value>".
if [[ -n "${FLINK_PROPERTIES}" ]]; then
  echo "Appending Flink properties to ${FLINK_CONF_FILE}: ${FLINK_PROPERTIES}"
  echo "" >>${FLINK_CONF_FILE}
  echo "# Extra properties." >>${FLINK_CONF_FILE}
  echo "${FLINK_PROPERTIES}" >>${FLINK_CONF_FILE}
fi

# Download remote job file.
if [[ -n "${FLINK_JOB_JAR_URI}" ]]; then
  download_job_file ${FLINK_JOB_JAR_URI} "${FLINK_HOME}/job/"
fi

if [[ -n "${FLINK_JOB_PYTHON_URI}" ]]; then
  download_job_file ${FLINK_JOB_PYTHON_URI} "${FLINK_HOME}/job/"
fi

if [[ -n "${FLINK_JOB_PYTHON_FILES_URI}" ]]; then
  download_job_file ${FLINK_JOB_PYTHON_FILES_URI} "${FLINK_HOME}/job/"
fi

# Handover to Flink base image's entrypoint.
exec "/docker-entrypoint.sh" "$@"
