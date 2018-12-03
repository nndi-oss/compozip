build:
	@go build -ldflags "-s -w" compozipd.go

clean:
	@rm compozipd
	@rm -rf ./uploads/**

docker: build
	sudo docker	build --rm -t=compozipd .

runTest:
	go run compozipd.go && curl -i -F "composer=@test_composer.json" --output uploads/test-vendor.zip -XPOST http://localhost:8080/vendor/test/zip

.default: build