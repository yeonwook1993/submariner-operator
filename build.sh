repo=$1
ver=$2
rm package/.image.submariner-operator bin/submariner-operator || true

make images
docker tag quay.io/submariner/submariner-operator:${ver} ${repo}/submariner-operator:${ver}
docker tag quay.io/submariner/submariner-operator:${ver}
docker push ${repo}/submariner-operator:${ver}

