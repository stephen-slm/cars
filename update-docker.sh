#! /usr/bin/env bash

RED="\e[31m"
GREEN="\e[32m"
ENDCOLOR="\e[0m"

for FILE in ./build/dockerfiles/*; do
  FILE_NAME=${FILE##*dockerfiles\/}
  LANG=${FILE_NAME%%.*}

  echo "${GREEN}Creating Docker Image${ENDCOLOR} - ${RED}${LANG}${ENDCOLOR}"

  if [ -z "$1" ]
    then docker build -f $FILE -t "virtual_machine_${LANG}" . > /dev/null
    else docker build --progress=plain -f $FILE -t "virtual_machine_${LANG}" .
  fi
  echo "${GREEN}Completed Building Docker Image${ENDCOLOR} - ${RED}${LANG}${ENDCOLOR}"
done

