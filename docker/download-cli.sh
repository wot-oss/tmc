#! /bin/sh

URL=$(curl -s https://api.github.com/repos/web-of-things-open-source/tm-catalog-cli/releases/latest | jq -r '.assets | .[] | select(.name == "tm-catalog-cli-linux-amd64") | .browser_download_url')

curl -OL $URL
mv ./tm-catalog-cli-linux-amd64 ./tm-catalog-cli
