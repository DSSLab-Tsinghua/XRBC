#!/bin/bash

# This script auto-build the repo and run the test for the claim 1 & 2
# ./autoBuildTest.bash

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Log functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to parse log files and write results
# Function to parse log files and write results
parse_log_and_write_results() {
    # Determine the result file path - always write to the script's directory
    local result_file=""
    if [[ $(pwd) == */bin ]]; then
        result_file="../resultE3.txt"
    else
        result_file="resultE3.txt"
    fi
    
    local current_date=$(date +%Y%m%d)
    local log_filename="${current_date}_Eva.log"
    
    log_info "Parsing log files from directories 4 to 94 for date: $current_date"
    log_info "Current directory: $(pwd)"
    log_info "Result file will be written to: $result_file"
    
    # Arrays to store values for averaging
    local total_latencies=()
    local cross_latencies=()
    local consensus_latencies=()
    local num_nodes_B=0
    local num_nodes_A=0
    local num_requests=0
    local valid_files=0
    
    # Determine the base path based on current directory
    local base_path=""
    if [[ $(pwd) == */bin ]]; then
        base_path="../"
    fi
    
    # Loop through directories 4 to 34
    for dir_num in {4..94}; do
        local log_path="${base_path}var/log/$dir_num/$log_filename"
        
        log_info "Checking log file: $log_path"
        
        # Check if the log file exists
        if [ ! -f "$log_path" ]; then
            log_warning "Log file not found: $log_path"
            continue
        fi
        
        # Read the first line with the pattern
        # --- THIS IS THE CORRECTED LINE ---
        local log_line=$(head -1 "$log_path" | grep -E "^[0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9]+, [0-9]+, [0-9]+, [0-9]+: [0-9]+-[0-9]+, [0-9]+$")
        
        if [ -z "$log_line" ]; then
            log_warning "No valid log entry found in first line of $log_path"
            continue
        fi
        
        log_info "Found log entry in $log_path: $log_line"
        
        # Extract values using regex
        if [[ $log_line =~ ^[0-9]{4}/[0-9]{2}/[0-9]{2}\ [0-9]{2}:[0-9]{2}:[0-9]{2}\ ([0-9]+),\ ([0-9]+),\ ([0-9]+),\ ([0-9]+):\ ([0-9]+)-([0-9]+),\ ([0-9]+)$ ]]; then
            local param1="${BASH_REMATCH[1]}"
            local param11="${BASH_REMATCH[2]}"
            local param2="${BASH_REMATCH[3]}"
            local total_latency="${BASH_REMATCH[4]}"
            local cross_latency="${BASH_REMATCH[5]}"
            local consensus_latency="${BASH_REMATCH[6]}"
            local param6="${BASH_REMATCH[7]}"
            
            # Store values for averaging
            total_latencies+=($total_latency)
            cross_latencies+=($cross_latency)
            consensus_latencies+=($consensus_latency)
            
            # Calculate number of nodes from first valid entry (should be same for all)
            if [ $valid_files -eq 0 ]; then
                num_nodes_B=$((param1 * 3 + 1))
                num_nodes_A=$((param11 * 3 + 1)) # Correctly use param11 for Group A
                num_requests=$param2
            fi
            
            valid_files=$((valid_files + 1))
            log_success "Successfully parsed $log_path"
            
        else
            log_error "Failed to parse log entry in $log_path: $log_line"
        fi
    done
    
    # Check if we found any valid files
    if [ $valid_files -eq 0 ]; then
        log_error "No valid log files found in any directory"
        return 1
    fi
    
    log_info "Found $valid_files valid log files"
    
    # Calculate averages
    local total_sum=0
    local cross_sum=0
    local consensus_sum=0
    
    for latency in "${total_latencies[@]}"; do
        total_sum=$((total_sum + latency))
    done
    
    for latency in "${cross_latencies[@]}"; do
        cross_sum=$((cross_sum + latency))
    done
    
    for latency in "${consensus_latencies[@]}"; do
        consensus_sum=$((consensus_sum + latency))
    done
    
    local avg_total=$((total_sum / valid_files))
    local avg_cross=$((cross_sum / valid_files))
    local avg_consensus=$((consensus_sum / valid_files))

    local num_nodes_B_E1=$(grep -oP 'Number Of Nodes \(Group B\):\s*\K\d+'  ../resultE1.txt)
    local avg_total_E1=$(grep -oP 'Average Total Latency\(ms\):\s*\K\d+' ../resultE1.txt)
    
    # Write results to file
    cat > "$result_file" << EOF
E3:
Number of Nodes (Group A): $num_nodes_A
Number of Request: $num_requests
Average Total Latency(ms) when Number of Nodes (Group B) is $num_nodes_B: $avg_total
Average Total Latency(ms) when Number of Nodes (Group B) is $num_nodes_B_E1: $avg_total_E1

EOF
    
    log_success "Results written to $result_file"s
    log_info "E2: Number Of Nodes: $num_nodes_B, Number Of Nodes (Group A): $num_nodes_A, Number of Request: $num_requests, Average Total Latency(ms): $avg_total, Average {Cross} Latency (ms): $avg_cross, Average Consensus in B Latency: $avg_consensus"
    log_info "Averages calculated from $valid_files log files"
    
    return 0
}

# Cleanup function
cleanup() {
    log_info "Cleaning up all processes..."
    
    # Terminate all server processes
    for i in {0..94}; do
        pkill -f "./server $i" 2>/dev/null
    done
    
    # Terminate all client processes
    pkill -f "./client" 2>/dev/null
    
    # Wait for processes to terminate completely
    sleep 2
    
    # Force kill if any processes remain
    pkill -9 -f "./server" 2>/dev/null
    pkill -9 -f "./client" 2>/dev/null
    
    log_success "All processes have been cleaned up"
}

# Set signal handlers
trap cleanup EXIT INT TERM
chmod +x ./bin/server ./bin/client

# Claim 1: Latency Breakdown: XRBC-wA (t=1, f=10)
# Step1: update the config

main() {
    log_info "Cleaning log files in last tests..."
    local log_dir="var/log"
    [[ -d "$log_dir" ]] && rm -rf "$log_dir"/* || log_warning "$log_dir does not exist or already empty"

    log_info "Starting autoBuildTest.bash execution"
    
    # Step 1: Call autoConfig.bash
    log_info "Step 1: Executing configuration script"
    if [ ! -f "./autoConfig.bash" ]; then
        log_error "autoConfig.bash file does not exist"
        exit 1
    fi
    
    chmod +x ./autoConfig.bash
    #t=1(m=4), f=30(n=91)
    ./autoConfig.bash 4 91 100 etc/conf.json
    
    if [ $? -ne 0 ]; then
        log_error "autoConfig.bash execution failed"
        exit 1
    fi
    log_success "Configuration script completed"
    
    # Step 2: Change to bin directory
    log_info "Step 2: Changing to bin directory"
    if [ ! -d "bin" ]; then
        log_error "bin directory does not exist"
        exit 1
    fi
    
    cd bin
    log_success "Changed to bin directory"
    
    # Step 3: Start 35 server nodes (./server 0 to ./server 35)
    log_info "Step 3: Starting server nodes (0-35)"
    
    # Check if server executable exists
    if [ ! -f "./server" ]; then
        log_error "server executable does not exist"
        exit 1
    fi
    
    # Start server nodes
    for i in {0..94}; do
        log_info "Starting server node $i"
        ./server $i &
        
        # Give each server some time to start
        sleep 0.2
    done
    
    log_success "All server nodes have been started"
    
    # Wait for server nodes to fully start
    log_info "Waiting for server nodes to fully start..."
    sleep 15    
    # Step 4: Start client node
    log_info "Step 4: Starting client node"
    
    # Check if client executable exists
    if [ ! -f "./client" ]; then
        log_error "client executable does not exist"
        exit 1
    fi
    
    # Start client
    log_info "Starting client: ./client 100 0 1 100"
    ./client 100 0 1 1 &
    CLIENT_PID=$!
    
    log_success "Client node has been started (PID: $CLIENT_PID)"
    
    # Wait for system to complete and check for Eva log files
    echo -e "${BLUE}============================================${NC}"
    log_info "🔍 WAITING FOR SYSTEM TO COMPLETE..."
    echo -e "${BLUE}============================================${NC}"
    
    sleep 90
    local max_wait_time=150
    local check_interval=30
    local elapsed_time=0
    local current_date=$(date +%Y%m%d)
    local eva_log_file="../var/log/4/${current_date}_Eva.log"
    
    log_info "Target Eva log file: $eva_log_file"
    
    while [ $elapsed_time -lt $max_wait_time ]; do
        if [ -f "$eva_log_file" ]; then
            echo -e "${GREEN}✅ SUCCESS: Eva log file found!${NC}"
            log_success "Eva log file found: $eva_log_file"
            break
        else
            echo -e "${YELLOW}⏳ WAITING: Eva log file not found yet... (${elapsed_time}s/${max_wait_time}s)${NC}"
            log_info "Checking again in $check_interval seconds..."
            sleep $check_interval
            elapsed_time=$((elapsed_time + check_interval))
        fi
    done
    
    if [ ! -f "$eva_log_file" ]; then
        echo -e "${RED}⚠️  TIMEOUT: Eva log file still not found after ${max_wait_time}s${NC}"
        log_warning "Proceeding with parsing anyway..."
    fi
    
    echo -e "${BLUE}============================================${NC}"
    
    # Process results before cleanup
    log_info "Processing results..."
    parse_log_and_write_results
    
    # Cleanup will be executed automatically in EXIT trap
    log_info "Test completed, cleaning up..."
}

# Check if script is being run correctly
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    main "$@"
fi



