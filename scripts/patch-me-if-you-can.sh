#!/bin/bash

set -xeuo pipefail

IFS=$'\n\t'

readonly USAGE="Usage: patch-me-if-you-can.sh --cluster <cluster-name> [ <component-name> ... ]"
readonly SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
readonly EIRINI_STAGING_BASEDIR=$(realpath "$SCRIPT_DIR/..")
readonly EIRINI_RELEASE_BASEDIR=$(realpath "$SCRIPT_DIR/../../eirini-release")


main() {
  check_flags "$@"
  shift 2

  export VALUES_FILE="$HOME/workspace/eirini-private-config/environments/kube-clusters/$(current_cluster_name)/values.yaml"
  if [ "$#" == "0" ]; then
    echo "No components specified. Nothing to do."
    exit 0
  fi

  local component img_id
  for component in "$@"; do
    echo "--- Patching component $component ---"
    docker_build "$component"
    docker_push "$component" "$img_id"
    update_values_file "$component" "$img_id"
  done

  helm_upgrade
  restart_eirini
}

check_flags() {
  if [ "$#" == "0" ]; then
    echo "$USAGE"
    exit 1
  fi

  if [ "$1" != "--cluster" ] && [ "$1" != "-c" ]; then
    echo "$USAGE"
    exit 1
  fi

  local cluster_name
  cluster_name=$2

  if [ -z "$cluster_name" ]; then
    echo "$USAGE"
    exit 1
  fi

  if [ "$cluster_name" != "$(current_cluster_name)" ]; then
    echo "Your current cluster is $(current_cluster_name), but you want to update $cluster_name. Please target $cluster_name"
    echo eval "\$(ibmcloud ks cluster config --export $cluster_name)"
    exit 1
  fi
}

docker_build() {
  local img_sha
  echo "Building docker image for $1"
  pushd "$EIRINI_STAGING_BASEDIR"
    img_sha=$(docker build -q . -f "$EIRINI_STAGING_BASEDIR/image/$component/Dockerfile" --build-arg GIT_SHA=big-sha)
    img_id=${img_sha#sha256:}
    docker tag $img_sha "eirini/recipe-$1:$img_id"
  popd
}

docker_push() {
  echo "Pushing docker image for $1"
  pushd "$EIRINI_STAGING_BASEDIR"
  docker push "eirini/recipe-$1:$2"
  popd
}

update_values_file() {
  echo "Applying docker image of $1 to kubernetes cluster"
  goml set --file $VALUES_FILE --prop "eirini.opi.staging.${1}_image_tag" --value "$2"
}

helm_upgrade() {
  pushd "$EIRINI_RELEASE_BASEDIR/helm/cf"
    helm dep update
  popd

  local secret ca_cert secret_name bits_tls_crt bits_tls_key
  secret=$(kubectl get pods --namespace uaa -o jsonpath='{.items[?(.metadata.name=="uaa-0")].spec.containers[?(.name=="uaa")].env[?(.name=="INTERNAL_CA_CERT")].valueFrom.secretKeyRef.name}')
  ca_cert=$(kubectl get secret "$secret" --namespace uaa -o jsonpath="{.data['internal-ca-cert']}" | base64 --decode -)

  secret_name=$(kubectl get secrets -n default | grep "$(current_cluster_name)" | cut -d ' ' -f 1)
  bits_tls_crt=$(kubectl get secret "$secret_name" --namespace default -o jsonpath="{.data['tls\.crt']}" | base64 --decode -)
  bits_tls_key=$(kubectl get secret "$secret_name" --namespace default -o jsonpath="{.data['tls\.key']}" | base64 --decode -)

  echo "HELM version: $(helm version)"
  pushd "$EIRINI_RELEASE_BASEDIR/helm"
    helm upgrade --install scf ./cf \
      --namespace scf \
      --values "${VALUES_FILE}" \
      --set "secrets.UAA_CA_CERT=${ca_cert}" \
      --set "bits.secrets.BITS_TLS_KEY=${bits_tls_key}" \
      --set "bits.secrets.BITS_TLS_CRT=${bits_tls_crt}"
  popd
}

current_cluster_name() {
  kubectl config current-context | cut -d / -f 1
}

restart_eirini(){
	local pod_name=$(kubectl get pods -n scf --selector=name=eirini --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
	kubectl delete pod $pod_name -n scf
}

main "$@"
