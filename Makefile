export REGISTRY ?= arnobkumarsaha
export JOB_NAME ?= kubernetes-go-test

push:
	@echo "Building project";\
	CGO_ENABLED=0 go build -o out;\
	docker build . -t $(REGISTRY)/$(JOB_NAME);\
	docker push $(REGISTRY)/$(JOB_NAME);\
	#kind load docker-image $(REGISTRY)/$(JOB_NAME)

apply:
	@kubectl delete jobs --ignore-not-found=true --wait=true $(JOB_NAME);\
	envsubst < job.template > job.yaml;\
	kubectl apply -f job.yaml;\

log:
	@kubectl wait --for=condition=complete --timeout=5m job/$(JOB_NAME);\
	printf "\nPod Logs:\n";\
	kubectl logs `kubectl get pods --selector=job-name=$(JOB_NAME) --output=jsonpath='{.items[*].metadata.name}'`

run: push apply log