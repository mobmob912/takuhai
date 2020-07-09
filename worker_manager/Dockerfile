FROM docker:stable-dind

RUN echo $(docker ps)

WORKDIR /worker_manager
ADD . /worker_manager

#ENV GOPATH /go

ENTRYPOINT ["./worker_manager"]
