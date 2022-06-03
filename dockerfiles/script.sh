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
  RUNTIME_START=$(date +%s%3N)
  $compiler "$sourceFile" - <"${stdInFile}"
  RUNTIME_END=$(date +%s%3N)
else
  COMPILE_START=$(date +%s%3N)
  $compiler "$sourceFile" "$additionalArguments"
  COMPILE_END=$(date +%s%3N)

  if [ $? -eq 0 ]; then
    RUNTIME_START=$(date +%s%3N)
    $output - <"${stdInFile}"
    RUNTIME_END=$(date +%s%3N)
  else
    echo "Compilation Failed"
  fi
fi


runtime=$(expr $RUNTIME_END - $RUNTIME_START)
compile=$(expr $COMPILE_END - $COMPILE_START)

echo "*-COMPILE::EOF-*" "$runtime" "$compile"
