.PHONY: all build run debug clean

all: build

build:
	docker build -t simulation .

run: build
	docker run -p 8081:80 simulation

debug: build
	docker run -p 8081:80 -e DEBUG=1 simulation

clean:
	docker rmi simulation
