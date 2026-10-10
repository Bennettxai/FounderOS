#!/bin/bash
# BusinessOS Terminal Init Script
# This file is sourced when the terminal starts (works in both bash and zsh)

# Colors
C_RESET='\033[0m'
C_BOLD='\033[1m'
C_DIM='\033[2m'
C_CYAN='\033[1;36m'
C_GREEN='\033[1;32m'
C_YELLOW='\033[1;33m'
C_RED='\033[1;31m'
C_MAGENTA='\033[1;35m'
C_BLUE='\033[1;34m'
C_WHITE='\033[1;37m'
C_GRAY='\033[0;90m'

# OSA CLI - 21-Agent Orchestration System
osa() {
  # Detect if running in container
  local IN_CONTAINER=false
  if [ -f /.dockerenv ] || grep -q docker /proc/1/cgroup 2>/dev/null; then
    IN_CONTAINER=true
  fi

  # API endpoints - use host.docker.internal in containers
  local BACKEND_API="http://localhost:8001"
  local OSA_API="http://localhost:3003"
  if [ "$IN_CONTAINER" = true ]; then
    BACKEND_API="http://host.docker.internal:8001"
    OSA_API="http://host.docker.internal:3003"
  fi

  local USER_ID="${BUSINESSOS_USER_ID}"

  # Check if user ID is available
  if [ -z "$USER_ID" ]; then
    printf "${C_RED}ERROR${C_RESET} Not authenticated (BUSINESSOS_USER_ID not set)\n"
    return 1
  fi

  case "$1" in
    ""|chat)
      # Interactive chat mode
      echo ""
      printf "${C_CYAN}${C_BOLD}"
      echo "  ╭──────────────────────────────────────╮"
      echo "  │           OSA TERMINAL               │"
      echo "  │     Type 'exit' to quit              │"
      echo "  ╰──────────────────────────────────────╯"
      printf "${C_RESET}"
      echo ""

      while true; do
        # Read user input with styled prompt
        printf "${C_MAGENTA}>${C_RESET} "
        read -r user_input

        # Check for exit
        case "$user_input" in
          exit|quit|q)
            echo ""
            printf "${C_DIM}Goodbye.${C_RESET}\n"
            echo ""
            return 0
            ;;
          "")
            continue
            ;;
          help|h|\?)
            echo ""
            printf "${C_CYAN}Commands:${C_RESET}\n"
            printf "  ${C_WHITE}any text${C_RESET}  ${C_DIM}Chat with OSA${C_RESET}\n"
            printf "  ${C_WHITE}exit${C_RESET}      ${C_DIM}Leave chat${C_RESET}\n"
            printf "  ${C_WHITE}clear${C_RESET}     ${C_DIM}Clear screen${C_RESET}\n"
            echo ""
            continue
            ;;
          clear)
            clear
            printf "${C_CYAN}OSA Terminal${C_RESET} ${C_DIM}(type 'exit' to quit)${C_RESET}\n"
            echo ""
            continue
            ;;
        esac

        printf "\n${C_YELLOW}...${C_RESET}\n\n"

        # Call the chat endpoint for conversations
        local response=$(curl -s "$OSA_API/api/chat" \
          -H "Content-Type: application/json" \
          -d "{\"message\":\"$user_input\",\"context\":{\"user_id\":\"$USER_ID\",\"platform\":\"businessos\"}}" 2>&1)

        # Check if successful and display response
        if echo "$response" | grep -q '"success":true'; then
          # Extract the response text using sed
          local raw_output=$(echo "$response" | sed 's/.*"response":"//' | sed 's/"[,}].*//')

          # Convert escape sequences properly
          local output=$(printf '%b' "$raw_output")

          if [ -n "$output" ]; then
            printf "${C_WHITE}%s${C_RESET}\n" "$output"
          else
            printf "${C_DIM}(No response)${C_RESET}\n"
          fi
          echo ""
        else
          local error=$(echo "$response" | grep -oE '"error":"[^"]*"' | head -1 | cut -d'"' -f4)
          printf "${C_RED}Error:${C_RESET} %s\n\n" "${error:-Request failed}"
        fi
      done
      ;;

    health)
      printf "${C_CYAN}Checking OSA...${C_RESET}\n"
      local result=$(curl -s "$OSA_API/health" 2>&1)
      if [ "$result" = "OK" ]; then
        printf "${C_GREEN}OK${C_RESET} OSA is running at ${C_DIM}%s${C_RESET}\n" "$OSA_API"
      else
        printf "${C_RED}ERROR${C_RESET} OSA not responding\n"
        printf "${C_DIM}Response: %s${C_RESET}\n" "$result"
      fi
      ;;

    agents)
      printf "${C_CYAN}${C_BOLD}Available Agents${C_RESET}\n"
      printf "${C_DIM}────────────────────────────────────${C_RESET}\n"
      local result=$(curl -s "$OSA_API/api/agents" 2>&1)
      if command -v jq >/dev/null 2>&1; then
        echo "$result" | jq -r '.[] | "  \(.type): \(.capabilities | join(", "))"' 2>/dev/null || echo "$result"
      else
        echo "$result"
      fi
      ;;

    generate|gen)
      if [ -z "$2" ]; then
        printf "${C_YELLOW}Usage:${C_RESET} osa gen ${C_DIM}<description>${C_RESET}\n"
        printf "${C_DIM}Example: osa gen \"task management system\"${C_RESET}\n"
        return 1
      fi

      local description="${*:2}"
      printf "\n${C_CYAN}${C_BOLD}Generating:${C_RESET} %s\n" "$description"
      printf "${C_DIM}────────────────────────────────────${C_RESET}\n\n"
      printf "${C_YELLOW}Running 21-agent orchestration...${C_RESET}\n\n"

      # Call the orchestrator directly
      local response=$(curl -s "$OSA_API/api/orchestrate" \
        -H "Content-Type: application/json" \
        -d "{\"prompt\":\"$description\",\"context\":{\"user_id\":\"$USER_ID\",\"platform\":\"businessos\"}}" 2>&1)

      # Check if successful
      if echo "$response" | grep -q '"success":true'; then
        printf "${C_GREEN}${C_BOLD}COMPLETE${C_RESET}\n\n"

        # Extract workflow ID
        local wf_id=$(echo "$response" | grep -oE '"workflow_id":"[^"]+"' | head -1 | cut -d'"' -f4)
        printf "${C_DIM}Workflow:${C_RESET} %s\n\n" "${wf_id:-N/A}"

        # Count successful agents
        local success_count=$(echo "$response" | grep -o '"success":true' | wc -l | tr -d ' ')
        printf "${C_CYAN}Agents:${C_RESET} %s executed\n" "$success_count"

        # Extract and show generated files
        local files=$(echo "$response" | grep -oE '"path":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$files" ]; then
          printf "\n${C_CYAN}Files:${C_RESET}\n"
          echo "$files" | while read -r file; do
            [ -n "$file" ] && printf "  ${C_DIM}-${C_RESET} %s\n" "$file"
          done
        fi

        printf "\n${C_GREEN}Done.${C_RESET} Check OSA workspace for files.\n\n"
      else
        printf "${C_RED}${C_BOLD}FAILED${C_RESET}\n\n"
        local error=$(echo "$response" | grep -oE '"error":"[^"]*"' | head -1 | cut -d'"' -f4)
        printf "${C_RED}Error:${C_RESET} %s\n\n" "${error:-Unknown error}"
      fi
      ;;

    status)
      if [ -z "$2" ]; then
        printf "${C_YELLOW}Usage:${C_RESET} osa status ${C_DIM}<app-id>${C_RESET}\n"
        return 1
      fi

      printf "${C_CYAN}Status:${C_RESET} %s\n" "$2"
      printf "${C_DIM}────────────────────────────────────${C_RESET}\n"

      local result=$(curl -s -H "X-User-ID: $USER_ID" "$BACKEND_API/api/internal/osa/status/$2" 2>&1)
      if command -v jq >/dev/null 2>&1; then
        echo "$result" | jq -r '
          if .status then
            "App ID:   \(.app_id // .appId)\n" +
            "Status:   \(.status)\n" +
            "Progress: \((.progress // 0) * 100)%"
          else
            "Error: " + (.error // .message // "Unknown error")
          end'
      else
        echo "$result"
      fi
      ;;

    list)
      printf "${C_DIM}Not yet available.${C_RESET}\n"
      printf "Use ${C_WHITE}osa agents${C_RESET} or ${C_WHITE}osa gen <desc>${C_RESET}\n"
      ;;

    help|--help|-h)
      echo ""
      printf "${C_CYAN}${C_BOLD}OSA CLI${C_RESET} ${C_DIM}- 21-Agent AI System${C_RESET}\n"
      printf "${C_DIM}────────────────────────────────────${C_RESET}\n\n"
      printf "${C_WHITE}osa${C_RESET}              ${C_DIM}Start chat${C_RESET}\n"
      printf "${C_WHITE}osa gen${C_RESET} <desc>   ${C_DIM}Generate code${C_RESET}\n"
      printf "${C_WHITE}osa agents${C_RESET}       ${C_DIM}List agents${C_RESET}\n"
      printf "${C_WHITE}osa health${C_RESET}       ${C_DIM}Check status${C_RESET}\n"
      printf "${C_WHITE}osa help${C_RESET}         ${C_DIM}Show this${C_RESET}\n"
      echo ""
      printf "${C_DIM}Examples:${C_RESET}\n"
      printf "  osa\n"
      printf "  osa gen \"user auth system\"\n"
      echo ""
      ;;

    *)
      printf "${C_RED}Unknown:${C_RESET} %s\n" "$1"
      printf "${C_DIM}Try 'osa help'${C_RESET}\n"
      return 1
      ;;
  esac
}

# Welcome message
echo ""
printf "${C_CYAN}${C_BOLD}BusinessOS Terminal${C_RESET}\n"
printf "${C_DIM}Type 'osa' to chat or 'osa help' for commands${C_RESET}\n"
echo ""
