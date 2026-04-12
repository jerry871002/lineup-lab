.PHONY: all build run clean

all: build

build:
	docker build -t stats .

run: build
	docker run -p 8082:80 stats

clean:
	docker rmi stats
