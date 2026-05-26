#!/bin/bash

kubectl delete pod proxy

docker build -t proxy:latest .

minikube image load proxy:latest

kubectl apply -f ../manifests/proxy.yaml