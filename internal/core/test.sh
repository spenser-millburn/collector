#!/bin/bash

# List of test functions to run
tests=(
  #    "TestNewConfigManager"
  #    "TestConfigManagerLifecycle"
  "TestLoadConfig"
  #    "TestSaveConfig"
  #    "TestGetConfig"
  #    "TestSetConfig"
  #    "TestWatchConfig"
)

# Run each test sequentially
for test in "${tests[@]}"; do
  echo "Running $test..."
  go test -run "^$test$"
  if [ $? -ne 0 ]; then
    echo "$test failed."
    exit 1
  fi
  echo "$test passed."
done

echo "All tests completed."
