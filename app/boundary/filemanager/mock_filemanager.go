package filemanager

//go:generate mockgen -destination=mock_filemanager_gen.go -package=filemanager github.com/wasya-io/go-kilo/app/boundary/filemanager FileManager