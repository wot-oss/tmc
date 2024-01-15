#! /bin/bash

# Parameters for this script, which are usually explicitly set in the pipeline:
#  
# FILENAME_PREFIX: "tm-catalog-cli" per default and normally set to the repo name in a pipeline
# GOOS_GARCH_TARGETS: list of of os/arch compile targets with defaults
   
filename_prefix=${FILENAME_PREFIX:-"tm-catalog-cli"}
targets=${GOOS_GARCH_TARGETS:-"linux,amd64;darwin,amd64;darwin,arm64;windows,amd64;windows,arm64"}

IFS=';' read -ra os_arch_array <<< $targets 

echo "building targets: $targets"
for os_arch in "${os_arch_array[@]}"; do
  export GOOS="${os_arch%%,*}"
  export GARCH="${os_arch#*,}"
  filename="${filename_prefix}-${GOOS}-${GARCH}"
  echo "compiling $filename"
  CGO_ENABLED=0 go build -o $filename
done
