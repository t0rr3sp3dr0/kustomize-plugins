#!/bin/sh

set -e

VERSION=$1
BUCKET_NAME=$2
RELEASE_URL=https://github.com/t0rr3sp3dr0/kustomize-plugins/releases/download/${VERSION}

TEMP_DIC=$(mktemp -d)
for KIND in ArgoCDProject ClusterRoles KustomizeBuild Namespace Template Unnamespaced IaC
do
	KIND_LOWERCASE=$(echo ${KIND} | tr '[:upper:]' '[:lower:]')
	wget -P $TEMP_DIC ${RELEASE_URL}/${KIND_LOWERCASE}-darwin-amd64
    wget -P $TEMP_DIC ${RELEASE_URL}/${KIND_LOWERCASE}-linux-amd64
done

aws s3 cp $TEMP_DIC s3://${BUCKET_NAME}/t0rr3sp3dr0/kustomize-plugins/${VERSION} --recursive

rm -r $TEMP_DIC