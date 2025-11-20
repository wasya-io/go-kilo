package mock_filemanager

//go:generate mockgen -destination=mock_filemanager_gen.go -package=mock_filemanager github.com/wasya-io/go-kilo/app/boundary/filemanager FileManager
