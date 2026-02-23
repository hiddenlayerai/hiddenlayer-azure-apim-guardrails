#!/usr/bin/env bash
set -euo pipefail

#############################################
# Load .env
#############################################
if [[ -f .env ]]; then
  set -a
  source .env
  set +a
else
  echo "❌ .env file not found"
  exit 1
fi

#############################################
# Validate required vars
#############################################
required_vars=(
  AZURE_RG
  APIM_NAME
  API_ID
  API_PATH
  API_DISPLAY_NAME
  OPENAI_API_KEY
)

for v in "${required_vars[@]}"; do
  if [[ -z "${!v:-}" ]]; then
    echo "❌ Missing required env var: $v"
    exit 1
  fi
done

#############################################
# Defaults
#############################################
OPENAI_BASE_URL="${OPENAI_BASE_URL:-https://api.openai.com}"
NAMED_VALUE_ID="${NAMED_VALUE_ID:-openai-api-key}"
NAMED_VALUE_DISPLAY="${NAMED_VALUE_DISPLAY:-OpenAI API Key}"
API_DESCRIPTION="${API_DESCRIPTION:-OpenAI passthrough via APIM}"
API_PROTOCOLS="${API_PROTOCOLS:-https}"
SUBSCRIPTION_REQUIRED="${SUBSCRIPTION_REQUIRED:-false}"
OPERATIONS_CSV="${OPERATIONS_CSV:-chat-completions|POST|/v1/chat/completions|Chat Completions,models|GET|/v1/models|List Models,embeddings|POST|/v1/embeddings|Embeddings,audio-speech|POST|/v1/audio/speech|Audio Speech}"

log() { echo "[$(date +'%H:%M:%S')] $*"; }

#############################################
# Named Value (secret)
#############################################
log "Ensuring named value..."
if az apim nv show -g "$AZURE_RG" -n "$APIM_NAME" --named-value-id "$NAMED_VALUE_ID" >/dev/null 2>&1; then
  az apim nv update \
    -g "$AZURE_RG" -n "$APIM_NAME" \
    --named-value-id "$NAMED_VALUE_ID" \
    --set value="$OPENAI_API_KEY" secret=true displayName="$NAMED_VALUE_DISPLAY" \
    >/dev/null
else
  az apim nv create \
    -g "$AZURE_RG" -n "$APIM_NAME" \
    --named-value-id "$NAMED_VALUE_ID" \
    --display-name "$NAMED_VALUE_DISPLAY" \
    --value "$OPENAI_API_KEY" \
    --secret true \
    >/dev/null
fi

#############################################
# API
#############################################
log "Ensuring API..."
if az apim api show -g "$AZURE_RG" -n "$APIM_NAME" --api-id "$API_ID" >/dev/null 2>&1; then
  az apim api update \
    -g "$AZURE_RG" -n "$APIM_NAME" \
    --api-id "$API_ID" \
    --set displayName="$API_DISPLAY_NAME" path="$API_PATH" description="$API_DESCRIPTION" \
    >/dev/null
else
  az apim api create \
    -g "$AZURE_RG" -n "$APIM_NAME" \
    --api-id "$API_ID" \
    --display-name "$API_DISPLAY_NAME" \
    --path "$API_PATH" \
    --protocols "$API_PROTOCOLS" \
    --service-url "$OPENAI_BASE_URL" \
    --subscription-required "$SUBSCRIPTION_REQUIRED" \
    --description "$API_DESCRIPTION" \
    >/dev/null
fi

#############################################
# Operations
#############################################
log "Ensuring operations..."
IFS=',' read -r -a OPS <<< "$OPERATIONS_CSV"
for op in "${OPS[@]}"; do
  IFS='|' read -r OP_ID METHOD URL_TEMPLATE OP_NAME <<< "$op"

  if az apim api operation show \
    -g "$AZURE_RG" -n "$APIM_NAME" --api-id "$API_ID" --operation-id "$OP_ID" \
    >/dev/null 2>&1; then
    az apim api operation update \
      -g "$AZURE_RG" -n "$APIM_NAME" --api-id "$API_ID" --operation-id "$OP_ID" \
      --set displayName="$OP_NAME" method="$METHOD" urlTemplate="$URL_TEMPLATE" \
      >/dev/null
  else
    az apim api operation create \
      -g "$AZURE_RG" -n "$APIM_NAME" --api-id "$API_ID" \
      --operation-id "$OP_ID" \
      --display-name "$OP_NAME" \
      --method "$METHOD" \
      --url-template "$URL_TEMPLATE" \
      >/dev/null
  fi
done

#############################################
# API Policy
#############################################
log "Applying API policy..."
SUB_ID="$(az account show --query id -o tsv)"
API_VERSION="2024-05-01"

POLICY_XML=$(cat <<EOF
<policies>
  <inbound>
    <base />
    <set-backend-service base-url="${OPENAI_BASE_URL}" />
    <!-- Avoid compressed backend responses so APIM policy fragments can safely read/inspect bodies -->
    <set-header name="Accept-Encoding" exists-action="override">
      <value>identity</value>
    </set-header>
    <set-header name="Authorization" exists-action="override">
      <value>Bearer {{${NAMED_VALUE_ID}}}</value>
    </set-header>
  </inbound>
  <backend><base /></backend>
  <outbound><base /></outbound>
  <on-error><base /></on-error>
</policies>
EOF
)

BODY=$(python3 - <<PY
import json
print(json.dumps({"properties":{"format":"rawxml","value":"""$POLICY_XML"""}}))
PY
)

az rest \
  --method PUT \
  --uri "https://management.azure.com/subscriptions/${SUB_ID}/resourceGroups/${AZURE_RG}/providers/Microsoft.ApiManagement/service/${APIM_NAME}/apis/${API_ID}/policies/policy?api-version=${API_VERSION}" \
  --headers "Content-Type=application/json" \
  --body "$BODY" \
  >/dev/null

log "✅ APIM OpenAI passthrough ready"
