#! /bin/bash

# Parameters for this script, which are usually explicitly set in the pipeline:
#
# FILENAME_PREFIX: "tmc" per default and normally set to the repo name in a pipeline


filename_prefix=${FILENAME_PREFIX:-"tmc"}
files=`find ./  -maxdepth 1 -name "${filename_prefix}*"`

echo -e "Used dependencies of compiled binaries\n${files}"

total=""
nl=$'\n'

for file in ${files}
do
  dep_info=$(go version -m ${file} | grep -o 'dep.*')

  readarray -t arr_dep_info <<< $dep_info

  for dep_line in "${arr_dep_info[@]}"
  do
    arr_dep_line=($dep_line)
    total+="${arr_dep_line[1]}@${arr_dep_line[2]}${nl}"
  done
done

echo "${total}" | sort -u