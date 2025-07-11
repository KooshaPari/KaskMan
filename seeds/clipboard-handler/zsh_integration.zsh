#!/bin/zsh
# ZSH Clipboard Image Handler Integration
# Part of KaskMan Autonomous R&D System - Seed Project
#
# This script integrates the clipboard handler into ZSH workflow
# and provides the foundation for autonomous learning and evolution.

# Configuration
KASKMAN_CLIPBOARD_HANDLER="${KASKMAN_HOME:-$HOME/.kaskman}/seeds/clipboard-handler/zsh_clipboard_handler.py"
KASKMAN_CONFIG_DIR="${KASKMAN_HOME:-$HOME/.kaskman}/clipboard"

# Ensure Python script is executable
if [[ -f "$KASKMAN_CLIPBOARD_HANDLER" ]]; then
    chmod +x "$KASKMAN_CLIPBOARD_HANDLER"
fi

# Main clipboard image processing function
kaskman_clipboard_process() {
    local verbose=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                verbose=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check if clipboard handler exists
    if [[ ! -f "$KASKMAN_CLIPBOARD_HANDLER" ]]; then
        echo "Error: Clipboard handler not found at $KASKMAN_CLIPBOARD_HANDLER" >&2
        return 1
    fi
    
    # Process clipboard
    local result
    if $verbose; then
        result=$(python3 "$KASKMAN_CLIPBOARD_HANDLER" --process --verbose 2>&1)
        local exit_code=$?
        echo "$result" >&2
    else
        result=$(python3 "$KASKMAN_CLIPBOARD_HANDLER" --process 2>/dev/null)
        local exit_code=$?
    fi
    
    if [[ $exit_code -eq 0 && -n "$result" ]]; then
        # Success: image was processed and stored
        echo "$result"
        
        # Auto-learn: record successful usage
        kaskman_record_usage "clipboard_process" "success"
        
        return 0
    else
        # No image or error
        if $verbose; then
            echo "No image found in clipboard or processing failed" >&2
        fi
        return 1
    fi
}

# Smart clipboard paste that automatically handles images
kaskman_smart_paste() {
    local target_location="${1:-.}"
    local verbose=false
    
    # Check if second argument is verbose flag
    if [[ "$2" == "-v" || "$2" == "--verbose" ]]; then
        verbose=true
    fi
    
    # Try to process clipboard image first
    local image_path
    if $verbose; then
        image_path=$(kaskman_clipboard_process --verbose)
    else
        image_path=$(kaskman_clipboard_process 2>/dev/null)
    fi
    
    if [[ $? -eq 0 && -n "$image_path" ]]; then
        # Image was processed, decide what to do with the path
        if [[ "$target_location" != "." ]]; then
            # If target location specified, copy or move file there
            local basename=$(basename "$image_path")
            local target_file="$target_location/$basename"
            
            if cp "$image_path" "$target_file" 2>/dev/null; then
                echo "$target_file"
                kaskman_record_usage "smart_paste" "copy_success"
            else
                echo "$image_path"
                kaskman_record_usage "smart_paste" "copy_failed"
            fi
        else
            # Just return the path
            echo "$image_path"
            kaskman_record_usage "smart_paste" "path_returned"
        fi
    else
        # No image, try regular clipboard paste
        if command -v pbpaste >/dev/null 2>&1; then
            pbpaste
        elif command -v xclip >/dev/null 2>&1; then
            xclip -selection clipboard -o
        else
            echo "No clipboard utility available" >&2
            return 1
        fi
        
        kaskman_record_usage "smart_paste" "text_paste"
    fi
}

# Show clipboard handler status and learning insights
kaskman_clipboard_status() {
    if [[ -f "$KASKMAN_CLIPBOARD_HANDLER" ]]; then
        echo "=== KaskMan Clipboard Handler Status ==="
        python3 "$KASKMAN_CLIPBOARD_HANDLER" --status
        
        # Show evolution status
        local evolution_file="$KASKMAN_CONFIG_DIR/evolution_ready.json"
        if [[ -f "$evolution_file" ]]; then
            echo ""
            echo "=== Evolution Status ==="
            echo "🚀 Ready for autonomous evolution!"
            echo "Signal file: $evolution_file"
        fi
    else
        echo "Clipboard handler not installed" >&2
        return 1
    fi
}

# Record usage for learning system
kaskman_record_usage() {
    local action="$1"
    local result="$2"
    local timestamp=$(date +%s)
    
    # Create usage log directory
    local usage_log_dir="$KASKMAN_CONFIG_DIR/usage_logs"
    mkdir -p "$usage_log_dir"
    
    # Log usage in simple format for learning system
    local log_file="$usage_log_dir/$(date +%Y-%m-%d).log"
    echo "$timestamp|$action|$result" >> "$log_file"
    
    # Keep only recent logs (last 30 days)
    find "$usage_log_dir" -name "*.log" -mtime +30 -delete 2>/dev/null
}

# Auto-detect friction points in ZSH usage
kaskman_detect_friction() {
    local command_history_file="$HISTFILE"
    local friction_log="$KASKMAN_CONFIG_DIR/friction_detected.log"
    
    if [[ -f "$command_history_file" ]]; then
        # Look for repetitive command patterns
        local recent_commands=$(tail -100 "$command_history_file" | cut -d';' -f2- 2>/dev/null || tail -100 "$command_history_file")
        
        # Detect clipboard-related friction
        local clipboard_commands=$(echo "$recent_commands" | grep -E "(pbpaste|pbcopy|xclip|clip)" | wc -l)
        if [[ $clipboard_commands -gt 3 ]]; then
            echo "$(date +%s)|command_repetition|clipboard_operations|$clipboard_commands" >> "$friction_log"
        fi
        
        # Detect build/test repetition
        local build_commands=$(echo "$recent_commands" | grep -E "(tsc|eslint|npm test|yarn test|go test)" | wc -l)
        if [[ $build_commands -gt 3 ]]; then
            echo "$(date +%s)|command_repetition|build_test_cycle|$build_commands" >> "$friction_log"
        fi
    fi
}

# ZSH hook to detect friction points
kaskman_preexec_hook() {
    # This runs before each command
    # Record command patterns for friction detection
    local cmd="$1"
    
    # Check for specific friction patterns
    case "$cmd" in
        *"tsc"*|*"eslint"*|*"npm test"*|*"yarn test"*)
            kaskman_record_usage "build_command" "executed"
            ;;
        *"pbpaste"*|*"pbcopy"*|*"xclip"*)
            kaskman_record_usage "clipboard_command" "executed"
            ;;
    esac
}

# ZSH hook to learn from command results
kaskman_precmd_hook() {
    # This runs after each command
    local exit_code=$?
    
    # Run friction detection periodically
    local last_friction_check_file="$KASKMAN_CONFIG_DIR/.last_friction_check"
    local current_time=$(date +%s)
    local last_check=0
    
    if [[ -f "$last_friction_check_file" ]]; then
        last_check=$(cat "$last_friction_check_file")
    fi
    
    # Check for friction every 5 minutes
    if [[ $((current_time - last_check)) -gt 300 ]]; then
        kaskman_detect_friction
        echo "$current_time" > "$last_friction_check_file"
    fi
}

# Install ZSH hooks
if [[ -z "$KASKMAN_HOOKS_INSTALLED" ]]; then
    # Add hooks to preexec and precmd arrays
    autoload -U add-zsh-hook
    add-zsh-hook preexec kaskman_preexec_hook
    add-zsh-hook precmd kaskman_precmd_hook
    
    export KASKMAN_HOOKS_INSTALLED=1
fi

# Convenient aliases
alias kcb='kaskman_clipboard_process'
alias kcb-status='kaskman_clipboard_status'
alias kpaste='kaskman_smart_paste'
alias kimg='kaskman_smart_paste'

# Smart clipboard binding (Ctrl+V enhancement)
# This replaces the default paste with smart paste
if [[ -o interactive ]]; then
    # Bind Ctrl+Shift+V to smart paste
    bindkey '^[^V' kaskman_smart_paste_widget
    
    # Widget function for ZLE
    kaskman_smart_paste_widget() {
        local result=$(kaskman_smart_paste)
        if [[ -n "$result" ]]; then
            LBUFFER+="$result"
        fi
        zle redisplay
    }
    zle -N kaskman_smart_paste_widget
fi

# Evolution trigger check
kaskman_check_evolution() {
    local evolution_file="$KASKMAN_CONFIG_DIR/evolution_ready.json"
    if [[ -f "$evolution_file" ]]; then
        echo ""
        echo "🧠 KaskMan Evolution Ready!"
        echo "Your clipboard handler is ready to evolve."
        echo "Run 'kaskman evolve clipboard-handler' to trigger autonomous evolution."
        echo ""
    fi
}

# Show evolution status on shell startup (once per day)
kaskman_startup_check() {
    local last_check_file="$KASKMAN_CONFIG_DIR/.last_evolution_check"
    local today=$(date +%Y-%m-%d)
    
    if [[ ! -f "$last_check_file" ]] || [[ "$(cat "$last_check_file" 2>/dev/null)" != "$today" ]]; then
        kaskman_check_evolution
        echo "$today" > "$last_check_file"
    fi
}

# Initialize on shell startup
if [[ -o interactive ]]; then
    # Run startup check in background to avoid slowing shell startup
    kaskman_startup_check &!
fi

# Export functions for use in other scripts
autoload -U kaskman_clipboard_process
autoload -U kaskman_smart_paste
autoload -U kaskman_clipboard_status

# Learning data export for KaskMan system
kaskman_export_learning_data() {
    local export_file="$KASKMAN_CONFIG_DIR/zsh_learning_export.json"
    local timestamp=$(date +%s)
    
    # Gather ZSH-specific learning data
    local zsh_data="{
        \"timestamp\": $timestamp,
        \"shell_info\": {
            \"version\": \"$ZSH_VERSION\",
            \"terminal\": \"$TERM\",
            \"user\": \"$USER\"
        },
        \"usage_patterns\": {
            \"session_count\": $(ls \"$KASKMAN_CONFIG_DIR/usage_logs/\"*.log 2>/dev/null | wc -l),
            \"friction_events\": $(wc -l < \"$KASKMAN_CONFIG_DIR/friction_detected.log\" 2>/dev/null || echo 0)
        },
        \"evolution_readiness\": $(test -f \"$KASKMAN_CONFIG_DIR/evolution_ready.json\" && echo true || echo false)
    }"
    
    echo "$zsh_data" > "$export_file"
    echo "Learning data exported to: $export_file"
}

# Manual evolution trigger
kaskman_force_evolution() {
    echo "🚀 Forcing evolution of clipboard handler..."
    
    # Create evolution signal
    local evolution_file="$KASKMAN_CONFIG_DIR/evolution_ready.json"
    local timestamp=$(date +%s)
    
    echo "{
        \"timestamp\": $timestamp,
        \"trigger\": \"manual_force\",
        \"phase_transition\": \"seed_to_growth\",
        \"user_initiated\": true
    }" > "$evolution_file"
    
    echo "Evolution signal created. KaskMan will process this in the next cycle."
    kaskman_export_learning_data
}

# Help function
kaskman_clipboard_help() {
    cat << 'EOF'
KaskMan Clipboard Handler - Autonomous Learning Seed

COMMANDS:
  kcb                  - Process clipboard image and return filepath
  kcb-status          - Show learning status and insights
  kpaste [location]   - Smart paste (handles images and text)
  kimg [location]     - Alias for kpaste

EVOLUTION:
  kaskman_force_evolution     - Force evolution to next phase
  kaskman_export_learning_data - Export learning data for analysis

The clipboard handler automatically learns from your usage patterns
and will evolve autonomously when conditions are met.

For more information: https://github.com/kaskman/autonomous-rd
EOF
}

# Make help available
alias kcb-help='kaskman_clipboard_help'

echo "✅ KaskMan Clipboard Handler initialized (Autonomous Learning Seed)"