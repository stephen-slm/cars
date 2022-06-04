#!/bin/bash

standard_out=$1

exec 1>"${standard_out}"

/runner
