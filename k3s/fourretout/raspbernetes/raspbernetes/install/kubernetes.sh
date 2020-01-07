#!/bin/bash
set -euo pipefail

kube_version="1.16.1-00"
kube_packages=("kubelet=${kube_version}" "kubectl=${kube_version}" "kubeadm=${kube_version}")

echo "Disabling swap and ensuring it doesn't turn back on after reboot"
dphys-swapfile swapoff
dphys-swapfile uninstall
update-rc.d dphys-swapfile remove
systemctl disable dphys-swapfile.service

# add repo list
curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat << EOF >> /etc/apt/sources.list.d/kubernetes.list
deb https://apt.kubernetes.io/ kubernetes-xenial main
EOF

# install required kubernetes packages
# NOTE: Put this in an until loop, because googles mirrors randomly are not found
# unsure if this has to do with too many concurrent hits from the same public IP
echo "Installing kubernetes ${kube_version}..."
until apt-get update; do echo "Retrying to update apt mirrors"; done
apt-get install -y --no-install-recommends "${kube_packages[@]}"
apt-mark hold "${kube_packages[@]}"

# pull down master images for faster build time in background
if [ "${KUBE_NODE_TYPE}" == "master" ]; then
  echo "Pulling down all kubeadm images..."
  kubeadm config images pull &
fi
