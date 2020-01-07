# K3s on TF Grid

## WHY

We want to be able to provision nodes with k3s installed and configured so that we can have a kubernetes cliusyer deployed easily within the grid

## HOW

A special reservation process will setup nodes with the correct binaries of [k3s](https://github.com/rancher/k3s) and with the proper initialisation so that

- We have the necessary kube config to access the cluster from a client. Indeed once k3s setup a k3s.yaml file will be generated on the master node and we need it to configure the cluster on a client machine (usually at ~/.kube/config)

- With kubectl installed on the client machine we can ask for the nodes of our cluster

```
$ kubectl get nodes
NAME        STATUS     ROLES    AGE   VERSION
zv2k8s-03   NotReady   <none>   20d   v1.16.3-k3s.2
zv2k8s-01   Ready      master   20d   v1.16.3-k3s.2
zv2k8s-04   Ready      <none>   20d   v1.16.3-k3s.2
zv2k8s-02   Ready      <none>   20d   v1.16.3-k3s.2
```

at this point we have a master nodes and worker nodes communicationg with each other.

## WHAT

**What do we have so far**

We have been able to deploy several containers including a drupal application connected to a mysql database [ressources files available /ressources/drupal-mysql](/ressources/drupal-mysql) and a wordpress with its mysql server [ressources files available /ressources/wordpress](/ressources/wordpress).

We also have deployed through HELM charts prometheus and grafana monitoring of the cluster

```
$ helm install --namespace mon --name prometheus  stable/prometheus-operator
$ helm list
NAME            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART
        APP VERSION
prometheus      mon             1               2019-12-17 18:13:08.296637106 +0100 CET deployed        prometheus-operator-8.3.3  0.34.0
```

We can deploy

- [x] k8s: Create and Deploy basic resources (Pods, multi container pods, deployment, replicaset, secrets, configmap )
- [x] k8s: Create Services with clusterIP and NodePort
- [x] k8s: Create PV and PVC with local path
- [x] k8s: Create and Deploy prometheus monitoring with helm

- [ ] k8s: Create and Deploy Storage solution (PV, PVC)
- [ ] k8s: Create and Deploy Ingress Controllers
- [ ] k8s: Create and Deploy an HA cluster
- [ ] k8s: Create and Deploy cert manager with helm
- [ ] k8s: Create and Deploy applications (CI/CD, multi tier)
- [ ] k8s: Test monitoring
- [ ] k8s: Test logging
- [ ] k8s: Test security
- [ ] k8s: Final report

**What is needed to have a production ready kubernetes cluster**

- networking
  - network policies
  - Test different CNI provider
  - ingress
  - automatic https certifcation with traeffik
- secrets
  - encryption at rest: Kubernetes API encrypts the secrets (optionally, using an external KMS system) before storing them in etcd.
- storage
  - decentralized storage
  - NFS
- high availability setup
  - HA PROXY and metalLB
