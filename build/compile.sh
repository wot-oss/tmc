#! /bin/bash

# Parameters for this script, which are usually explicitly set in the pipeline:
#  
# FILENAME_PREFIX: "tm-catalog-cli" per default and normally set to the repo name in a pipeline
# GOOS_GARCH_TARGETS: list of of os/arch compile targets with defaults
# TMC_VERSION: version to be set in source code variable cmd/TmcVersion when compile
   
filename_prefix=${FILENAME_PREFIX:-"tm-catalog-cli"}
targets=${GOOS_GARCH_TARGETS:-"linux,amd64;darwin,amd64;darwin,arm64;windows,amd64;windows,arm64"}
version=${TMC_VERSION:-"n/a"}

IFS=';' read -ra os_arch_array <<< $targets 

echo "building targets: $targets"
for os_arch in "${os_arch_array[@]}"; do
  export GOOS="${os_arch%%,*}"
  export GARCH="${os_arch#*,}"
  if [ "${GOOS}" == "windows" ]; then
      export EXT=".exe"
  else
      export EXT=""
  fi
  filename="${filename_prefix}-${GOOS}-${GARCH}${EXT}"
  echo "compiling $filename"
  CGO_ENABLED=0 go build -o $filename \
  -ldflags="-X github.com/web-of-things-open-source/tm-catalog-cli/cmd.TmcVersion=${version}"
done
