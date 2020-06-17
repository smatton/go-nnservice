From golang:buster

RUN apt-get update && apt-get install -y g++

ENV CGO_CXXFLAGS=-std=c++11


RUN go get github.com/evan176/hnswgo

WORKDIR /go/src/github.com/evan176/hnswgo

RUN make
