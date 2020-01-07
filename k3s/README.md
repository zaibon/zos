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

## Storage

Rook is an open source cloud-native storage orchestrator, providing the platform, framework, and support for a diverse set of storage solutions to natively integrate with cloud-native environments.

Rook turns storage software into self-managing, self-scaling, and self-healing storage services. It does this by automating deployment, bootstrapping, configuration, provisioning, scaling, upgrading, migration, disaster recovery, monitoring, and resource management. Rook uses the facilities provided by the underlying cloud-native container management, scheduling and orchestration platform to perform its duties.

installing [rook](https://rook.io/docs/rook/v1.2/)

```
git clone --single-branch --branch release-1.2 https://github.com/rook/rook.git
cd cluster/examples/kubernetes/ceph
kubectl create -f common.yaml
kubectl create -f operator.yaml
kubectl create -f cluster-test.yaml
```

### Installing [rook NFS](https://rook.io/docs/rook/v1.2/nfs.html)

NFS allows remote hosts to mount filesystems over a network and interact with those filesystems as though they are mounted locally. This enables system administrators to consolidate resources onto centralized servers on the network.

#### First deploy the Rook NFS operator using the following commands:

```
$ cd resources/storage/rook-NFS
$ kubectl create -f 1-operator.yaml
```

We will create a NFS server instance that exports storage that is backed by the default StorageClass. In k3s environments storageClass "local-path": Only support ReadWriteOnce access mode so for the PVC taht must be created before creating NFS CRD instance.

```
$ kubectl create -f 2-nfs.yaml
$  kubectl get pvc -n rook-nfs
NAME                STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
nfs-default-claim   Bound    pvc-9804c4f1-80e1-45ec-bf05-3dbdd012564e   1Gi        RWO            local-path     3m49s
$ kubectl get po -n rook-nfs
NAME         READY   STATUS    RESTARTS   AGE
rook-nfs-0   1/1     Running   0          16s
```

#### Accessing the Export through dynamic NFS provisioning

Once the NFS Operator and an instance of NFSServer is deployed. A storageclass has to be created to dynamically provisioning volumes.
The StorageClass need to have the following 3 parameters passed.

- exportName: It tells the provisioner which export to use for provisioning the volumes.
- nfsServerName: It is the name of the NFSServer instance.
- nfsServerNamespace: It namespace where the NFSServer instance is running.

```
$ kubectl create -f 3-sc.yaml
```

Once the above storageclass has been created create a PV claim referencing the storageclass as shown in the example given below.

```
$ kubectl create -f 4-pvc.yaml
```

#### Consuming the Export

Now we can consume the PV that we just created by creating an example web server app that uses the above `PersistentVolumeClaim` to claim the exported volume. There are 2 pods that comprise this example:

- A web server pod that will read and display the contents of the NFS share
- A writer pod that will write random data to the NFS share so the website will continually update
  Start both the busybox pod (writer) and the web server from the ressources/storage/rook-NFS folder:

```
kubectl create -f busybox-rc.yaml
kubectl create -f web-rc.yaml
```

CANT CREATE POD SUCK IN CONTAINER CREATING

```
  Warning  FailedMount       15m                   kubelet, zv2k8s-04  Unable to attach or mount volumes: unmounted volumes=[rook-nfs-vol], unattached volumes=[default-token-f6sx2 rook-nfs-vol]: timed out waiting for the condition
  Warning  FailedMount       2m31s (x10 over 16m)  kubelet, zv2k8s-04  MountVolume.SetUp failed for volume "pvc-903d405d-0c9c-4af7-bc2d-356fc03905fb" : mount failed: exit status 255
Mounting command: mount
Mounting arguments: -t nfs 10.43.102.171:/nfs-default-claim /var/lib/kubelet/pods/91d477ac-7e3f-4957-b427-0a3f2a68847b/volumes/kubernetes.io~nfs/pvc-903d405d-0c9c-4af7-bc2d-356fc03905fb
```

probably need nfs-common

### Installing [rook CEPH](https://rook.io/docs/rook/v1.2/ceph.html)
