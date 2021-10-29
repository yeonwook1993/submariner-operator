set -e
_REPO=$1
_VER=$2
rm package/.image.submariner-operator bin/submariner-operator || true
make images
docker build -t $_REPO/submariner-operator:$_VER -f package/Dockerfile.submariner-operator .
docker push $_REPO/submariner-operator:$_VER
