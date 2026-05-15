#!/usr/bin/env bash
set -euo pipefail

awslocal sqs create-queue --queue-name order-events-dlq
DLQ_ARN=$(awslocal sqs get-queue-attributes \
  --queue-url http://localstack:4566/000000000000/order-events-dlq \
  --attribute-names QueueArn \
  --query 'Attributes.QueueArn' --output text)

awslocal sqs create-queue --queue-name order-events \
  --attributes "{\"RedrivePolicy\":\"{\\\"deadLetterTargetArn\\\":\\\"${DLQ_ARN}\\\",\\\"maxReceiveCount\\\":\\\"3\\\"}\"}"
