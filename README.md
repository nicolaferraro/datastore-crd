# Datastore CRD

This repository implements a controller for Datastore resources. A Datastore is a StatefulSet with additional hooks
to make it easier to implement common workflows for stateful applications.

## Running

**Prerequisite**: Kubernetes cluster version should be greater than 1.9.

```sh
# assumes you have a working kubeconfig, not required if operating in-cluster
$ go run *.go -kubeconfig=$HOME/.kube/config -logtostderr

# create a CustomResourceDefinition
$ kubectl create -f artifacts/datastore-crd.yaml

# create a custom resource of type Foo
$ kubectl create -f artifacts/examples/example-datastore.yaml

# check statefulsets created through the custom resource
$ kubectl get statefulset
```

## Installing in Kubernetes

```sh
# Connect to docker daemon
# eval $(minikube docker-env)

# build the docker image
docker build -t datastore-controller .

# create the controller in Kubernetes
kubectl create -f artifacts/datastore-controller.yaml -n kube-system
```
