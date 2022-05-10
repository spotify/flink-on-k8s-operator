/*
Copyright 2020 Google LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package flinkcluster

// This script is part of the cluster's ConfigMap and is mounted into the
// job (submitter) container at `/opt/flink-operator/submit-job.sh` for job
// submission.
var submitJobScript = `
#! /usr/bin/env bash

# This script checks the status of the JobManager and submits the Flink job when ready.
# When job submission is finished, the result is recorded in the pod termination log in the following YAML format.
#
# When succeeded:
#
# jobID: ec74209eb4e3db8ae72db00bd7a830aa
# message: |
#   Successfully submitted!
#   /opt/flink/bin/flink run --jobmanager flinkjobcluster-sample-jobmanager:8081 --class org.apache.flink.streaming.examples.wordcount.WordCount --parallelism 2 --detached ./examples/streaming/WordCount.jar --input ./README.txt
#	Starting execution of program
#	Printing result to stdout. Use --output to specify output path.
#	Job has been submitted with JobID ec74209eb4e3db8ae72db00bd7a830aa
#
# When submission fails (no jobID):
#
# message: |
#   Aborted submit because JobManager is unavailable.
#   Checking job manager to be ready. Will check success of 2 API calls for stable job submission.
#	curl -sS "http://t3-jobmanager:8081/jobs"
#	curl: (7) Failed to connect to t3-jobmanager port 8081: Connection refused
#   ...

set -euo pipefail

readonly TERM_LOG="/dev/termination-log"
TERM_LOG_SIZE=0
TERM_LOG_MAX_SIZE=4096

function echo_log() {
    local msg="$1"
    local log_file="$2"
    echo -e "${msg}" | tee -a "${log_file}"
}

function write_term_log() {
    local msg=$1
    local size=$(echo -e "${msg}" | wc -c)

    if ((TERM_LOG_SIZE + size > TERM_LOG_MAX_SIZE)); then
        return 0
    fi

    echo "${msg}" >>"${TERM_LOG}"
    TERM_LOG_SIZE=$((TERM_LOG_SIZE + size))

    return $((size))
}

# write message to termination log as YAML format.
function write_term_log_msg() {
    local result_msg="$1"
    local log_file="$2"

    # Write result message.
    write_term_log "message: |"
    write_term_log "  ${result_msg}"

    # Append submit log to message.
    # Two space indentation is required to write strings in the form of YAML literal block scalar.
    IFS=''
    while read -r line; do
        # Insert indentation before printing line.
        write_term_log "  ${line}"
        if [ $? = 0 ]; then
            break
        fi
    done <"${log_file}"
}

function check_jm_ready() {
    # Waiting for 10 mins.
    local -r MAX_RETRY=120
    local -r RETRY_INTERVAL=5s
    local -r REQUIRED_SUCCESS_NUMBER=3
    local -r CONNECT_TIMEOUT=5
    local success_count=0

    echo_log "Checking job manager to be ready. Will check success of ${REQUIRED_SUCCESS_NUMBER} API calls for stable job submission." "job_check_log"
    for ((i = 1; i <= MAX_RETRY; i++)); do
        echo_log "curl -sS \"http://${FLINK_JM_ADDR}/jobs\"" "job_check_log"
        if curl -sS --connect-timeout ${CONNECT_TIMEOUT} "http://${FLINK_JM_ADDR}/jobs" 2>&1 | tee -a job_check_log; then
            ((success_count++))
            echo_log "\nSuccess ${success_count}/${REQUIRED_SUCCESS_NUMBER}" "job_check_log"
            if ((success_count < REQUIRED_SUCCESS_NUMBER)); then
                echo_log "\nWaiting..." "job_check_log"
                sleep "${RETRY_INTERVAL}"
                continue
            fi
            echo_log "\nJob manager is ready now. Tried ${i} time(s), every ${RETRY_INTERVAL} and succeeded ${success_count} time(s)." "job_check_log"
            return 0
        else
            echo_log "\nWaiting..." "job_check_log"
        fi
        sleep "${RETRY_INTERVAL}"
    done

    echo_log "\nReached max retry count(${MAX_RETRY}) to check job manager status." "job_check_log"
    echo_log "Aborted to submit job." "job_check_log"

    write_term_log_msg "Aborted submit because JobManager is unavailable." "job_check_log"

    return 1
}

function submit_job() {
    local job_id=""

    # Submit job and extract the job ID
    echo "/opt/flink/bin/flink run $*" | tee -a submit_log
    /opt/flink/bin/flink run "$@" 2>&1 | tee -a submit_log
    local -r job_exit_code=$?
    local -r job_id_indicator="Job has been submitted with JobID"
    job_id=$(grep "${job_id_indicator}" submit_log | awk -F "${job_id_indicator}" '{printf $2}' | awk '{printf $1}')

    # Write result as YAML format to pod termination-log.
    # On failure, write log only.
    if [[ -z ${job_id} ]]; then
		if [ $job_exit_code -eq 0 ]; then
			write_term_log_msg "The submitter finished successfully but there is no job id. Did you forget to execute the pipeline?" "submit_log"
			return 1
		fi        
        write_term_log_msg "Failed to submit." "submit_log"
        return 1
    fi

    # write job ID if there is one
    write_term_log "jobID: ${job_id}"

    # check the job's exit code
    if [ $job_exit_code -ne 0 ]; then
        write_term_log_msg "Job failed with a non-zero exit code: ${job_exit_code}" "submit_log"
        return 1
    fi        

    # On success, write log
    write_term_log_msg "Successfully submitted!" "submit_log"
    return 0
}

function main() {
    echo -e "---------- Checking job manager status ----------"
    if ! check_jm_ready; then
        exit 1
    fi

    echo -e "\n---------- Submitting job ----------"
    if ! submit_job "$@"; then
        exit 2
    fi
}

main "$@"
`
