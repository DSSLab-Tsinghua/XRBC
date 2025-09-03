#!/bin/bash
# Usage: ./autoConfig.bash <num_replicas_shard0> <num_replicas_shard1> <consensus> <config_file>

# ------------------------------------------------------------------
# Helper: emit JSON array of replica objects
# ------------------------------------------------------------------
generate_replicas() {
    local num=$1      # how many replicas
    local start_id=$2 # first id to assign
    local port_base=$3
    local out=""

    for ((i=0; i<num; i++)); do
        out+=$(cat <<EOF
        {
          "id": "$((start_id+i))",
          "host": "localhost",
          "port": "$((port_base+i))"
        }
EOF
        )
        if (( i < num-1 )); then
            out+=","
        fi
    done
    printf "%s" "$out"
}

# ------------------------------------------------------------------
# Parse arguments
# ------------------------------------------------------------------
if [[ $# -ne 4 ]]; then
    echo "Usage: $0 <num_replicas_shard0> <num_replicas_shard1> <consensus> <config_file>"
    exit 1
fi

NUM_SHARD0=$1
NUM_SHARD1=$2
CONSENSUS=$3
CONFIG_FILE=$4

# ------------------------------------------------------------------
# Build new config fragments
# ------------------------------------------------------------------
SHARD0_JSON=$(generate_replicas "$NUM_SHARD0" 0 11000)
SHARD1_JSON=$(generate_replicas "$NUM_SHARD1" "$NUM_SHARD0" $((11000 + NUM_SHARD0)))

# ------------------------------------------------------------------
# Patch the file atomically
# ------------------------------------------------------------------
jq \
  --argjson consensus "$CONSENSUS" \
  --argjson shard0 "[$SHARD0_JSON]" \
  --argjson shard1 "[$SHARD1_JSON]" \
  '
  .consensus = $consensus |
  .replicas[0].config = $shard0 |
  .replicas[1].config = $shard1
  ' "$CONFIG_FILE" > "tmp.$$.json" && mv "tmp.$$.json" "$CONFIG_FILE"

echo "Configuration updated:"
echo "  - Shard 0 replicas: $NUM_SHARD0  (ids 0..$((NUM_SHARD0-1)))"
echo "  - Shard 1 replicas: $NUM_SHARD1  (ids $NUM_SHARD0..$((NUM_SHARD0+NUM_SHARD1-1)))"
echo "  - Consensus: $CONSENSUS"