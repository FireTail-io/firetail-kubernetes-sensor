.PHONY: build
build:
	GOARCH=amd64 GOOS=linux docker build . -t firetail/kubernetes-sensor

.PHONY: publish
run: build
	docker run -p 8080:8080 firetail/kubernetes-sensor

.PHONY: publish
publish: build
	aws ecr get-login-password --region eu-west-1 | docker login --username AWS --password-stdin 799185653336.dkr.ecr.eu-west-1.amazonaws.com
	docker tag firetail/kubernetes-sensor:latest 799185653336.dkr.ecr.eu-west-1.amazonaws.com/firetail/kubernetes-sensor:latest
	docker push 799185653336.dkr.ecr.eu-west-1.amazonaws.com/firetail/kubernetes-sensor:latest
