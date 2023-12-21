# Module Build Information / Commands



# Go Build

```
go build -ldflags '-X main.Version=0.4.0 -extldflags "-static"' -o ./bin/objectfilter ./src
```


# Canon Build

```
canon -arch "arm64" go build -ldflags '-X main.Version=0.5.0' -o ./bin/objectfilter ./src

```

Module Upload

```
viam module upload --version X.X.X --platform e.g. darwin/arm64 bin/objectfilter
```


