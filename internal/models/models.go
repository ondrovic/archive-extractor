package models

import (
	"archive/zip"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode"
)

type ArchiveFile interface {
	Name() string
	HeaderName() string
}

type ZipFile struct {
	*zip.File
}

func (f ZipFile) Name() string {
	return f.File.Name
}

func (f ZipFile) HeaderName() string {
	return f.File.FileHeader.Name
}

type RarFile struct {
	*rardecode.FileHeader
}

type SevenZipFile struct {
	*sevenzip.File
}

func (f RarFile) Name() string {
	return f.FileHeader.Name
}

func (f RarFile) HeaderName() string {
	return f.FileHeader.Name // RAR doesn't have a separate header name
}


func (f SevenZipFile) Name() string {
	return f.File.Name
}

func (f SevenZipFile) HeaderName() string {
	return f.File.Name // 7zip doesn't have a separate header name
}