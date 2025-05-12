include .env

.PHONY: build
build:
	docker build -f build_setup/Dockerfile . --platform linux/amd64 -t firetail/kubernetes-sensor

.PHONY: publish
publish: build
	aws ecr get-login-password --region eu-west-1 | docker login --username AWS --password-stdin 799185653336.dkr.ecr.eu-west-1.amazonaws.com
	docker tag firetail/kubernetes-sensor:latest 799185653336.dkr.ecr.eu-west-1.amazonaws.com/firetail/kubernetes-sensor:${VERSION}
	docker push 799185653336.dkr.ecr.eu-west-1.amazonaws.com/firetail/kubernetes-sensor:${VERSION}

.PHONY: build-dev
build-dev:
	docker build -f build_setup/Dockerfile . -t firetail/kubernetes-sensor-dev

.PHONY: publish
dev: build-dev
	docker run -it \
		-p 8080:80 \
		-e FIRETAIL_API_URL=https://api.logging.eu-west-1.sandbox.firetail.app/logs/bulk \
		-e FIRETAIL_API_TOKEN=${FIRETAIL_API_TOKEN} \
		-e FIRETAIL_KUBERNETES_SENSOR_DEV_MODE=true \
		-e FIRETAIL_KUBERNETES_SENSOR_DEV_SERVER_ENABLED=true \
		-e DISABLE_SERVICE_IP_FILTERING=true \
		-e ENABLE_ONLY_LOG_JSON=true \
		firetail/kubernetes-sensor-dev
