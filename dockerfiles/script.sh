#!/bin/bash

compiler=$1
sourceFile=$2
stdInFile=$3
output=$4
additionalArguments=$5
standard_out=$6
standard_error_out=$7

exec 1>"${standard_out}"
exec 2>"${standard_error_out}"

RUNTIME_START=0
RUNTIME_END=0

COMPILE_START=0
COMPILE_END=0

if [ "$output" = "" ]; then
  RUNTIME_START=$(date +%s.%2N)
  $compiler "$sourceFile" - <"${stdInFile}"
  RUNTIME_END=$(date +%s.%2N)
else
  COMPILE_START=$(date +%s.%2N)
  $compiler "$sourceFile" "$additionalArguments"
  COMPILE_END=$(date +%s.%2N)

  if [ $? -eq 0 ]; then
    RUNTIME_START=$(date +%s.%2N)
    $output - <"${stdInFile}"
    RUNTIME_END=$(date +%s.%2N)
  else
    echo "Compilation Failed"
  fi
fi


runtime=$(echo "$RUNTIME_END - $RUNTIME_START" | bc)
compile=$(echo "$COMPILE_END - $COMPILE_START" | bc)

echo "*-COMPILE::EOF-*" "$runtime" $"$compile"