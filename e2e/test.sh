#!/bin/bash

# in order to run test correctly set proper k8s config path for KUBECONFIG env
KUBECONFIG=/root/.kube/config
# add correct path to kubectl
PATH=/usr/local/bin:$PATH
#set namespace
NAMESPACE=default
# new file test
echo "running  new file test"
POD=$(kubectl get pod -l app=nginx-app -o jsonpath="{.items[0].metadata.name}")
kubectl -n "$NAMESPACE" exec -it "$POD" -- touch /usr/bin/newfile

sleep 4
# file deleted test
echo "running  removing file"
POD=$(kubectl get pod -l app=nginx-app -o jsonpath="{.items[0].metadata.name}")
kubectl -n "$NAMESPACE" exec -it "$POD" -- rm -f /usr/bin/tr


sleep 4
# file changed test
echo "running  file changed test"
POD=$(kubectl get pod -l app=nginx-app -o jsonpath="{.items[0].metadata.name}")
kubectl -n "$NAMESPACE" exec -it "$POD" -- cp /usr/bin/cut /usr/bin/tr
