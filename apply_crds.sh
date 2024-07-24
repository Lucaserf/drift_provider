#! bin/bash


#delete all childs
kubectl get ctrldrift -o json | kubectl delete -f -

#crds and provider
kubectl apply -f package/crds/


#provider config
kubectl apply -f examples/provider/config.yaml
kubectl apply -f deploy_crossplane.yaml

kubectl apply -f examples/sample/ctrldrift.yaml

