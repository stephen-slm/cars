#! /usr/bin/env bash

RED="\e[31m"
GREEN="\e[32m"
ENDCOLOR="\e[0m"

for FILE in ./build/dockerfiles/*; do
  echo "${GREEN}Creating Docker Image${ENDCOLOR} - ${RED}${FILE##*dockerfiles\/}${ENDCOLOR}"

  if [ -z "$1" ]
    then
      docker build -f $FILE -t "virtual_machine_${FILE##*dockerfiles\/Dockerfile.}" . > /dev/null
    else
      docker build --progress=plain -f $FILE -t "virtual_machine_${FILE##*dockerfiles\/Dockerfile.}" .
  fi
  echo "${GREEN}Completed Building Docker Image${ENDCOLOR} - ${RED}${FILE##*dockerfiles\/}${ENDCOLOR}"
done

