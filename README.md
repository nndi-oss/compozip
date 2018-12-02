Compozip
========

Download `zip` or `tar`composer vendor directories.

This is intended to be used as a tutorial to get people into
Go and building services other than just CRUD apps..

Basically, it's a simple API with one endpoint:

```
POST /vendor/{your-project-name}/{extension}
```

### Start the Server

```
$ compozipd -u PATH_TO_UPLOADS_DIRECTORY
```

### Download with CURL

```sh
curl -i -F "composer=@test_composer.json" --output test-vendor.zip -XPOST http://localhost:8080/vendor/test/zip
```

### Download with `compozip` CLI

```sh
$ ./compozip -port 8080 -c test_composer.json
Uploading test_composer.json ...
Downloading vendor archive (vendor.zip)...
Downloaded vendor archive to vendor.zip

$ ls
test_composer.json  vendor.zip.
```

# LICENSE

MIT

----

Copyright (c) 2018, Zikani Nyirenda Mwase