build:
	@go build -ldflags "-s -w" -o dist/compozipd

clean:
	@rm -rf ./dist 
	@rm -rf ./uploads/**

docker: build
	sudo docker	build --rm -t=nndi-oss/compozip .

runTest:
	go run compozipd.go && curl -i -F "composer=@test_composer.json" --output uploads/test-vendor.zip -XPOST http://localhost:8080/vendor/zip

.default: build