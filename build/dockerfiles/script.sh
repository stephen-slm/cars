#!/bin/bash

input=$1
standard_out=$2
standard_error_out=$3

exec 1>"${standard_out}"
exec 2>"${standard_error_out}"

python3 main.py "$input"
