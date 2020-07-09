.PHONY: docker-build docker-run docker-clean docker-build-run

docker-build:
	-docker rm mobmob912/takuhai-worker-manager
	docker build -t mobmob912/takuhai-worker-manager .


WORKER_NAME=dind
HOST_IP=192.168.3.29
docker-run: docker-clean
	docker run --name takuhai-worker-manager --privileged --add-host=host:${HOST_IP} -p 4871:4871 mobmob912/takuhai-worker-manager --name ${WORKER_NAME} --masterAddr=http://host:3000 --workerAddr=http://0.0.0.0:4871

docker-clean:
	-docker kill takuhai-worker-manager
	-docker rm takuhai-worker-manager

docker-build-run: docker-clean docker-build docker-run
