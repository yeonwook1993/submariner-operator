set -e

rm package/.image.submariner-operator bin/submariner-operator || true

make images
docker build -t yeonwook1993/submariner-operator:devel -f package/Dockerfile.submariner-operator .
docker push yeonwook1993/submariner-operator:devel
chown -R classact /home/classact/submariner-operator
