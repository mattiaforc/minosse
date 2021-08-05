default: run

compile:
	./compile_alpine.sh ..

build: compile
	docker build -f Dockerfile ../ -t minosse

run: compile build
	docker container rm minosse-bench || true
	docker run -d --name minosse-bench -p 8080:8080 minosse
	sleep 1
	./benchmark.sh