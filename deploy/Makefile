install:
	kustomize build --enable-helm --load-restrictor=LoadRestrictionsNone . | kubectl apply -f -
rm:
	kustomize build --enable-helm --load-restrictor=LoadRestrictionsNone . | kubectl delete -f -
