docker-worker:
	@docker build -t workflow-poc-worker -f Dockerfile-worker .

docker-tag-worker: docker-worker
	@docker tag workflow-poc-worker us.gcr.io/trading-dev-201715/workflow-poc-worker

docker-push-worker: docker-tag-worker
	@docker push  us.gcr.io/trading-dev-201715/workflow-poc-worker

docker-server:
	@docker build -t workflow-poc-server -f Dockerfile-server .

docker-tag-server: docker-server
	@docker tag workflow-poc-server us.gcr.io/trading-dev-201715/workflow-poc-server

docker-push-server: docker-tag-server
	@docker push  us.gcr.io/trading-dev-201715/workflow-poc-server
