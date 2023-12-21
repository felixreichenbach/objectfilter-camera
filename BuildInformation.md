# Module Build Information / Commands



Build Binary

```
go build -ldflags '-X main.Version=0.4.0 -extldflags "-static"' -o ./bin/objectfilter ./src



# works on rapi
canon -arch "arm64" go build -ldflags '-X main.Version=0.5.0' -o ./bin/objectfilter ./src

```


Module Upload

```
viam module upload --version 0.4.0 --platform darwin/arm64 bin/objectfilter
```


