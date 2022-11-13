go build -gcflags="all=-N -l" -o temp/api cmd/services/cars-api/main.go
go build -gcflags="all=-N -l" -o temp/loader cmd/services/cars-loader/main.go
