
.PHONY: build
build:
	docker build -t gateway .

.PHONY: build-app
build-app:
	docker build -f Dockerfile.app -t app .

.PHONY: run
run:
	docker run --rm --name gateway -p 3030:3000 -p 5050:5000 gateway

# make APP_ID=1 run-app
.PHONY: run-app
run-app:
	docker run --rm --name app-${APP_ID} -e APP_ID=${APP_ID} -p 800${APP_ID}:8000 app


