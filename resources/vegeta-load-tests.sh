#!/bin/bash
DEFAULT_EP_RATE="50"
DEFAULT_EP_DURATION="10"
DEFAULT_TMC_SERVICE_URL="http://localhost:8080"
DEFAULT_WORKERS=30
DEFAULT_ENDPOINTS="inventory,repos,authors,manufacturers,mpns,healthz"

EP_RATE=$DEFAULT_EP_RATE
EP_DURATION=$DEFAULT_EP_DURATION
TMC_SERVICE_URL=$DEFAULT_TMC_SERVICE_URL
WORKERS=$DEFAULT_WORKERS
ENDPOINTS=()
BEARER_TOKEN=""

show_usage() {
  echo "Usage: $0 [OPTIONS]"
  echo "Options:"
  echo "  -r, --rate <RATE>        Set the attack rate (e.g., '10'). Default: ${DEFAULT_EP_RATE}"
  echo "  -d, --duration <DURATION> Set the attack duration (e.g., '30'). Default: ${DEFAULT_EP_DURATION}"
  echo "  -u, --url <URL>          Set the base URL of the TMC service (e.g., 'http://localhost:8080'). Default: ${DEFAULT_TMC_SERVICE_URL}"
  echo "  -w, --workers <WORKERS>  Set the number of workers for Vegeta (e.g., '50'). Default: ${DEFAULT_WORKERS}"
  echo "  -e, --endpoints <ENDPOINTS> Comma-separated list of endpoints to test (e.g., 'repos,authors'). Default: ${DEFAULT_ENDPOINTS}"
  echo "  -t, --token <TOKEN>      Bearer token for authentication (optional)"
  echo "  -h, --help               Display this help message and exit."
  echo ""
  echo "Example:"
  echo "  $0 -r 100 -d 20 -u http://my-tmc-service:8080 -t YOUR_BEARER_TOKEN"
  exit 0
}

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -r|--rate)
      EP_RATE="$2"
      shift
      ;;
    -d|--duration)
      EP_DURATION="$2"
      shift
      ;;
    -u|--url)
      TMC_SERVICE_URL="$2"
      shift
      ;;
    -w|--workers)
      WORKERS="$2"
      shift
      ;;
    -e|--endpoints)
      # Split the comma-separated string into an array
      IFS=',' read -r -a ENDPOINTS <<< "$2"
      shift
      ;;
    -t|--token)
      BEARER_TOKEN="$2"
      shift
      ;;
    -h|--help)
      show_usage
      ;;
    *)
      echo "Unknown parameter: $1"
      show_usage
      ;;
  esac
  shift
done

# Use the default endpoints if none were provided
if [ ${#ENDPOINTS[@]} -eq 0 ]; then
  IFS=',' read -r -a ENDPOINTS <<< "$DEFAULT_ENDPOINTS"
fi

# export PATH=$PATH:$(go env GOPATH)/bin # Uncomment if you need to ensure PATH here

mkdir -p tests

REPORT_FILE="tests/report_rate_${EP_RATE}_workers_${WORKERS}.txt"
RESULTS_PREFIX="tests/results_rate_${EP_RATE}_workers_${WORKERS}"

> "${REPORT_FILE}"

echo "Starting API Performance Tests..."
echo "----------------------------------------------------" | tee -a "${REPORT_FILE}"
echo "Base Service URL: ${TMC_SERVICE_URL}" | tee -a "${REPORT_FILE}"
echo "Attack Rate: ${EP_RATE}" | tee -a "${REPORT_FILE}"
echo "Attack Duration: ${EP_DURATION}" | tee -a "${REPORT_FILE}"
echo "Number of Workers: ${WORKERS}" | tee -a "${REPORT_FILE}"
if [ -n "$BEARER_TOKEN" ]; then
  echo "Authentication: Bearer token provided" | tee -a "${REPORT_FILE}"
else
  echo "Authentication: None" | tee -a "${REPORT_FILE}"
fi
echo "----------------------------------------------------" | tee -a "${REPORT_FILE}"
echo "" | tee -a "${REPORT_FILE}"

for EP_PATH in "${ENDPOINTS[@]}"; do
  echo "----------------------------------------------------" | tee -a "${REPORT_FILE}"
  echo "Testing Endpoint: ${TMC_SERVICE_URL}/${EP_PATH}" | tee -a "${REPORT_FILE}"
  echo "Rate: ${EP_RATE}, Duration: ${EP_DURATION}, Workers: ${WORKERS}" | tee -a "${REPORT_FILE}"
  echo "----------------------------------------------------" | tee -a "${REPORT_FILE}"
  echo "" | tee -a "${REPORT_FILE}"

  if [ -n "$BEARER_TOKEN" ]; then
    echo "GET ${TMC_SERVICE_URL}/${EP_PATH}" | \
      vegeta attack \
        -rate "${EP_RATE}/s" \
        -duration="${EP_DURATION}s" \
        -workers ${WORKERS} \
        -timeout=1000s \
        -header "Authorization: Bearer ${BEARER_TOKEN}" | \
      tee "${RESULTS_PREFIX}_${EP_PATH//\//_}.bin" | \
      vegeta report -type text >> "${REPORT_FILE}"
  else
    echo "GET ${TMC_SERVICE_URL}/${EP_PATH}" | \
      vegeta attack \
        -rate "${EP_RATE}/s" \
        -duration="${EP_DURATION}s" \
        -workers ${WORKERS} \
        -timeout=1000s | \
      tee "${RESULTS_PREFIX}_${EP_PATH//\//_}.bin" | \
      vegeta report -type text >> "${REPORT_FILE}"
  fi
  
  echo "" | tee -a "${REPORT_FILE}" 
  echo "Completed test for ${EP_PATH}."
done

echo "----------------------------------------------------" | tee -a "${REPORT_FILE}"
echo "Load Tests Completed." | tee -a "${REPORT_FILE}"
echo "Report available in '${REPORT_FILE}'." | tee -a "${REPORT_FILE}"
echo "Raw results are in '${RESULTS_PREFIX}_*.bin' files." | tee -a "${REPORT_FILE}"
echo "----------------------------------------------------" | tee -a "${REPORT_FILE}"