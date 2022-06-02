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

START=$(date +%s.%2N)

if [ "$output" = "" ]; then
  $compiler "$sourceFile" - <"${stdInFile}"
else
  $compiler "$sourceFile" "$additionalArguments"

  if [ $? -eq 0 ]; then
    $output - <"${stdInFile}"
  else
    echo "Compilation Failed"
  fi
fi


END=$(date +%s.%2N)
runtime=$(echo "$END - $START" | bc)

echo "*-COMPILE::EOF-*" "$runtime"