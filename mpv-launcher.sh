#!/bin/bash

# This script helps launch MPV inside a Docker container
# It sets up a virtual framebuffer if needed and handles environment variables

# Check if we're running inside Docker
if [ -f /.dockerenv ]; then
  # Set up a virtual framebuffer if not displaying to a real screen
  if [ -z "$DISPLAY" ]; then
    export DISPLAY=:99
    Xvfb $DISPLAY -screen 0 1920x1080x24 &
    xvfb_pid=$!
    sleep 1
  fi
fi

# MPV arguments
MPV_ARGS="--no-terminal"

# Launch MPV with all arguments passed to this script
exec mpv $MPV_ARGS "$@"
