#!/bin/bash

# k6 Performance Results Analyzer
# Compares current test results against baseline
# Exits with code 1 if performance degraded by > 20%

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DEGRADATION_THRESHOLD=20  # Percentage
BASELINE_FILE=""
CURRENT_FILE=""

# Usage information
usage() {
    echo "Usage: $0 <baseline.json> <current.json> [options]"
    echo ""
    echo "Options:"
    echo "  --threshold <N>    Set degradation threshold (default: 20%)"
    echo "  --html             Generate HTML report"
    echo "  --verbose          Show detailed metrics"
    echo ""
    echo "Example:"
    echo "  $0 baseline/baseline-osa.json results/current-osa.json --threshold 15"
    exit 1
}

# Parse arguments
if [ $# -lt 2 ]; then
    usage
fi

BASELINE_FILE="$1"
CURRENT_FILE="$2"
shift 2

VERBOSE=false
GENERATE_HTML=false

while [ $# -gt 0 ]; do
    case "$1" in
        --threshold)
            DEGRADATION_THRESHOLD="$2"
            shift 2
            ;;
        --html)
            GENERATE_HTML=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Validate files exist
if [ ! -f "$BASELINE_FILE" ]; then
    echo -e "${RED}Error: Baseline file not found: $BASELINE_FILE${NC}"
    exit 1
fi

if [ ! -f "$CURRENT_FILE" ]; then
    echo -e "${RED}Error: Current results file not found: $CURRENT_FILE${NC}"
    exit 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is required but not installed${NC}"
    echo "Install with: brew install jq (macOS) or apt install jq (Linux)"
    exit 1
fi

echo "========================================================"
echo "  📊 k6 Performance Analysis"
echo "========================================================"
echo ""
echo -e "${BLUE}Baseline:${NC} $BASELINE_FILE"
echo -e "${BLUE}Current:${NC}  $CURRENT_FILE"
echo -e "${BLUE}Threshold:${NC} ${DEGRADATION_THRESHOLD}% degradation"
echo ""

# Extract metrics using jq
extract_metric() {
    local file=$1
    local metric=$2
    local percentile=$3

    if [ -z "$percentile" ]; then
        # Extract simple metric (e.g., http_req_failed rate)
        jq -r ".metrics.\"$metric\".values.rate // .metrics.\"$metric\".values.avg // 0" "$file"
    else
        # Extract percentile metric (e.g., http_req_duration p(95))
        jq -r ".metrics.\"$metric\".values.\"$percentile\" // 0" "$file"
    fi
}

# Calculate percentage change
calc_change() {
    local baseline=$1
    local current=$2

    if [ "$baseline" = "0" ] || [ -z "$baseline" ]; then
        echo "N/A"
        return
    fi

    echo "scale=2; (($current - $baseline) / $baseline) * 100" | bc
}

# Compare metrics
compare_metric() {
    local name=$1
    local metric=$2
    local percentile=$3
    local lower_is_better=$4  # true/false

    local baseline_val=$(extract_metric "$BASELINE_FILE" "$metric" "$percentile")
    local current_val=$(extract_metric "$CURRENT_FILE" "$metric" "$percentile")

    if [ "$baseline_val" = "0" ] && [ "$current_val" = "0" ]; then
        return
    fi

    local change=$(calc_change "$baseline_val" "$current_val")

    # Format output
    printf "%-30s" "$name:"

    if [ "$baseline_val" != "N/A" ]; then
        printf " Baseline: %10s" "$baseline_val"
    fi

    if [ "$current_val" != "N/A" ]; then
        printf " → Current: %10s" "$current_val"
    fi

    if [ "$change" != "N/A" ]; then
        # Determine if change is good or bad
        local is_degradation=false

        if [ "$lower_is_better" = true ]; then
            # For metrics where lower is better (latency, errors)
            if (( $(echo "$change > $DEGRADATION_THRESHOLD" | bc -l) )); then
                is_degradation=true
            fi
        else
            # For metrics where higher is better (success rate)
            if (( $(echo "$change < -$DEGRADATION_THRESHOLD" | bc -l) )); then
                is_degradation=true
            fi
        fi

        # Color-code the change
        if [ "$is_degradation" = true ]; then
            printf " ${RED}(%+.1f%%)${NC}\n" "$change"
            return 1
        elif (( $(echo "$change < 0" | bc -l) )) && [ "$lower_is_better" = true ]; then
            printf " ${GREEN}(%+.1f%%)${NC}\n" "$change"
            return 0
        elif (( $(echo "$change > 0" | bc -l) )) && [ "$lower_is_better" = false ]; then
            printf " ${GREEN}(%+.1f%%)${NC}\n" "$change"
            return 0
        else
            printf " ${YELLOW}(%+.1f%%)${NC}\n" "$change"
            return 0
        fi
    else
        printf "\n"
        return 0
    fi
}

# Track if any degradation detected
DEGRADATION_DETECTED=false

# Analyze key metrics
echo "📈 Response Time Analysis:"
echo "----------------------------------------"
compare_metric "P50 Latency" "http_req_duration" "p(50)" true || DEGRADATION_DETECTED=true
compare_metric "P95 Latency" "http_req_duration" "p(95)" true || DEGRADATION_DETECTED=true
compare_metric "P99 Latency" "http_req_duration" "p(99)" true || DEGRADATION_DETECTED=true
compare_metric "Avg Latency" "http_req_duration" "" true || DEGRADATION_DETECTED=true
echo ""

echo "⚠️  Error Rate Analysis:"
echo "----------------------------------------"
compare_metric "HTTP Failures" "http_req_failed" "" true || DEGRADATION_DETECTED=true
compare_metric "Error Rate" "error_rate" "" true || DEGRADATION_DETECTED=true
echo ""

echo "✅ Success Rate Analysis:"
echo "----------------------------------------"
compare_metric "Success Rate" "success_rate" "" false || DEGRADATION_DETECTED=true
echo ""

# Verbose mode: Show custom metrics
if [ "$VERBOSE" = true ]; then
    echo "📊 Custom Metrics:"
    echo "----------------------------------------"

    # OSA-specific metrics
    if jq -e '.metrics.osa_generate_latency' "$CURRENT_FILE" > /dev/null 2>&1; then
        compare_metric "OSA Generate P95" "osa_generate_latency" "p(95)" true
        compare_metric "OSA Status P95" "osa_status_latency" "p(95)" true
        compare_metric "OSA Orchestrate P95" "osa_orchestrate_latency" "p(95)" true
    fi

    # Hybrid architecture metrics
    if jq -e '.metrics.direct_path_latency' "$CURRENT_FILE" > /dev/null 2>&1; then
        compare_metric "Direct Path P95" "direct_path_latency" "p(95)" true
        compare_metric "CoT Path P95" "cot_path_latency" "p(95)" true
        compare_metric "Routing Overhead P95" "routing_overhead_ms" "p(95)" true
    fi

    echo ""
fi

# Extract request counts
baseline_reqs=$(jq -r '.metrics.http_reqs.values.count // 0' "$BASELINE_FILE")
current_reqs=$(jq -r '.metrics.http_reqs.values.count // 0' "$CURRENT_FILE")

echo "📦 Request Volume:"
echo "----------------------------------------"
printf "Baseline Requests: %s\n" "$baseline_reqs"
printf "Current Requests:  %s\n" "$current_reqs"
echo ""

# Generate HTML report (optional)
if [ "$GENERATE_HTML" = true ]; then
    echo "📄 Generating HTML report..."

    REPORT_FILE="performance-comparison-$(date +%Y%m%d-%H%M%S).html"

    cat > "$REPORT_FILE" <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>Performance Comparison Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        table { border-collapse: collapse; width: 100%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
        .degraded { background-color: #ffcccc; }
        .improved { background-color: #ccffcc; }
        .stable { background-color: #ffffcc; }
    </style>
</head>
<body>
    <h1>k6 Performance Comparison Report</h1>
    <p><strong>Generated:</strong> $(date)</p>
    <p><strong>Baseline:</strong> $BASELINE_FILE</p>
    <p><strong>Current:</strong> $CURRENT_FILE</p>
    <p><strong>Threshold:</strong> ${DEGRADATION_THRESHOLD}% degradation</p>

    <h2>Summary</h2>
    <table>
        <tr>
            <th>Metric</th>
            <th>Baseline</th>
            <th>Current</th>
            <th>Change (%)</th>
            <th>Status</th>
        </tr>
        <tr>
            <td>P95 Latency</td>
            <td>$(extract_metric "$BASELINE_FILE" "http_req_duration" "p(95)")</td>
            <td>$(extract_metric "$CURRENT_FILE" "http_req_duration" "p(95)")</td>
            <td>$(calc_change "$(extract_metric "$BASELINE_FILE" "http_req_duration" "p(95)")" "$(extract_metric "$CURRENT_FILE" "http_req_duration" "p(95)")")</td>
            <td>$([ "$DEGRADATION_DETECTED" = true ] && echo "DEGRADED" || echo "OK")</td>
        </tr>
    </table>
</body>
</html>
EOF

    echo -e "${GREEN}✅ HTML report generated: $REPORT_FILE${NC}"
fi

# Final verdict
echo "========================================================"
if [ "$DEGRADATION_DETECTED" = true ]; then
    echo -e "${RED}❌ PERFORMANCE REGRESSION DETECTED${NC}"
    echo ""
    echo "Performance has degraded by more than ${DEGRADATION_THRESHOLD}%."
    echo "Please investigate and optimize before merging."
    echo ""
    exit 1
else
    echo -e "${GREEN}✅ PERFORMANCE WITHIN ACCEPTABLE RANGE${NC}"
    echo ""
    echo "No significant performance regression detected."
    echo ""
    exit 0
fi
