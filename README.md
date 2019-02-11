Compozip
========

Compozip is a Server (and Client) for downloading packaged composer vendor folders.

Upload your `composer.json` or `composer.lock` and the server will take of downloading the 
dependencies, zip them up and give you a nice downloadable archive.

Vendor archives can be downloaded as either `zip` or `tar` archives.

## API

Basically, it's a simple API with one endpoint. It also has an [OpenAPI specification](./swagger.yaml) ;P

```
POST /vendor/{extension}

Content-Type: multipart/form-data

... multipart-boundary stuff
composer=FILE_DATA
```

## Downloading Vendor archives

You may be thinking it's too much work to download from that API endpoint.
Fortunately, you don't have to code to download from the API endpoint.

You have at least two options, illustrated below:

### 1. Download with `compozip` CLI

```sh
$ ./compozip -host "localhost" -port 8080 -f composer.json
Uploading composer.json ...
Downloading vendor archive (vendor.zip)...
Downloaded vendor archive to vendor.zip

$ ls
composer.json  vendor.zip
```

### 2. Download with CURL

With a `composer.json` file:

```sh
$ curl -F "composer=@composer.json" --output test-vendor.zip -XPOST http://localhost:8080/vendor/zip
```

Or with a `composer.lock` file:

```
$ curl -F "composer=@composer.lock" --output test-vendor.zip -XPOST http://localhost:8080/vendor/zip
```

<small>NOTE: You should also be able to use a tool like [Postman](https://getpostman.com) 
or create your own HTML form ([See example in index.html](./index.html))</small>

## Archives generated from `composer.lock` files

When you upload a `composer.lock` file, you get repeatability in the downloaded archive since the 
dependencies in a lock file are locked to specific versions. This is great for repeatable builds!

> **NOTE**: The downloaded zip or tar archive generated from a `composer.lock`
> will contain a stub `composer.json`. It is only there to make the backend processing faster.
> Make sure you do not overwrite your original `composer.json` with this file.

The stub `composer.json` has the following contents:

```json
{
	"name": "compozip/generated",
	"description": "This is a stub composer.json generated because you uploaded a composer.lock file. Please discard it and use your original composer.json.",
	"license": "MIT",
	"require": {
		"php":">=5.6.30"
	}
}
```

## Building the code

You need to have [Go 1.11 (or greater)](https://golang.org), [PHP](https://php.net) and [Composer](https://getcomposer.org) installed.

You MUST build the code before starting the server.

```sh
$ git clone https://github.com/zikani03/compozipd.git
$ cd compozipd

# Build the Server
$ go build 

# Build the CLI client
$ go build cmd/compozip/compozip.go
```

## Starting the Server

### Run the server binary directly

```
$ compozipd -u ./uploads -h "localhost:8080"
```

### Run the server via Docker

This project comes with a [Dockerfile](./Dockerfile) for running the Server in
a Docker container. 

Use the following commands to build and start the Docker container.

```sh
$ sudo docker	build --rm -t=compozipd .
$ sudo docker	run --rm --publish=80:80 -it compozipd
```

## What problem does it solve/address?

Unfortunately downloading composer dependencies sometimes takes a long time 
since Composer has to do multiple HTTP/Git requests.

The idea is to put this on a server with fast internet in the Cloud to download
the dependencies, so you just download the archive which _should be_ faster 
since it's just one HTTP request instead of bajillions.

## Caveats

* There is no guarantee that downloading the vendor archive will be faster than running
`composer install` on your machine as composer most likely has a cache on your machine if
you use it often.

* This won't play nicely with private repositories/composer packages. We don't have
a good solution for that yet. Sorry.

* The Docker image/container uses [php:7.2-fpm-alpine](https://github.com/docker-library/php/blob/b99209cc078ebb7bf4614e870c2d69e0b3bed399/7.2/alpine3.8/fpm/Dockerfile) 
image from [Dockerhub](https://hub.docker.com/_/php/) and as such may NOT contain
all the PHP extensions that dependencies in your `composer.json` or `composer.lock` require.
We might create a base Docker image that contains MOST of the common 
PHP extensions to prevent this from happening.

* This is NOT YET _production_ ready.

## Contributing

**Contributions are very WELCOME here!**

File an Issue or send a Pull Request if you find a bug or have a feature idea to improve this project.

## LICENSE

MIT

----

Copyright (c) 2018, NNDI