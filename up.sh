#!/bin/bash

# delete everything pre-existing
kubectl delete -f ./manifests/proxy.yaml --ignore-not-found
kubectl delete -f ./manifests/prometheus.yaml --ignore-not-found
kubectl delete -f ./manifests/grafana.yaml --ignore-not-found
kubectl delete -f ./manifests/grafana-dashboards-configmap.yaml --ignore-not-found

#create namespace for prometheus and grafana
kubectl create namespace monitoring

# recreate grafana dashboards configmap
kubectl create configmap grafana-dashboards   --from-file=proxy-dashboard.json=./manifests/grafana_dashboards_settings/dashboards.json   --namespace monitoring   --dry-run=client -o yaml > manifests/grafana-dashboards-configmap.yaml


#create proxy
cp ~/.minikube/profiles/minikube/client.crt proxy/cert/
cp ~/.minikube/profiles/minikube/client.key proxy/cert/

docker build -t proxy:latest ./proxy

minikube image load proxy:latest

kubectl apply -f manifests/proxy.yaml

# create prometheus
kubectl apply -f manifests/prometheus.yaml


# create grafana
kubectl apply -f manifests/grafana-dashboards-configmap.yaml # dashboards in json
kubectl apply -f manifests/grafana.yaml

# wait for pods to be ready
kubectl wait --for=condition=ready pod -l app=proxy -n default --timeout=60s
kubectl wait --for=condition=ready pod -l app=prometheus -n monitoring --timeout=60s
kubectl wait --for=condition=ready pod -l app=grafana -n monitoring --timeout=60s

# portforwarding for proxy to be reachable by external elements from the cluster (I'm using the operator outside the cluster using go run)
kubectl port-forward -n monitoring svc/prometheus 9090:9090 &
kubectl port-forward -n monitoring svc/grafana 3000:3000 &
kubectl port-forward -n default svc/proxy 8080:8080

