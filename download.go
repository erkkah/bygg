package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

func (b *bygge) handleDownload(target string, url string, checksum ...string) error {
	if !(strings.HasSuffix(url, ".tar") || strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, "tgz")) {
		return fmt.Errorf("Unsupported file: %v", url)
	}

	b.verbose("Downloading %s", url)
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}

	targetDate := getFileDate(target).In(time.FixedZone("GMT", 0))
	if !targetDate.IsZero() {
		req.Header.Set("If-Modified-Since", targetDate.Format(time.RFC1123))
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotModified {
		b.verbose("%s unmodified, skipping download", url)
		return nil
	}

	modified := response.Header.Get("Last-Modified")
	var modificationDate time.Time
	if modified != "" {
		modificationDate, err = time.Parse(time.RFC1123, modified)
		if err != nil {
			modificationDate = time.Time{}
		}
	}

	if err = os.MkdirAll(target, 0771); err != nil {
		return err
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), target)
	if err != nil {
		return err
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err = io.Copy(tmpFile, response.Body); err != nil {
		return err
	}
	_, _ = tmpFile.Seek(0, 0)

	if len(checksum) > 0 {
		ok, err := validateChecksum(tmpFile, checksum[0])
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("checksum verification failed for %q", url)
		}
		_, _ = tmpFile.Seek(0, 0)
	}

	var reader io.Reader = tmpFile
	if strings.HasSuffix(url, "gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return err
		}
	}

	err = unpackArchive(target, reader)
	if err != nil {
		return err
	}

	if !modificationDate.IsZero() {
		_ = os.Chtimes(target, modificationDate, modificationDate)
	}

	return nil
}

func validateChecksum(file *os.File, checksum string) (bool, error) {
	if !strings.HasPrefix(checksum, "md5:") {
		return false, fmt.Errorf("checksum must start with \"md5:\"")
	}

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, err
	}
	sum := fmt.Sprintf("md5:%x", hash.Sum(nil))
	return sum == checksum, nil
}

func unpackArchive(target string, source io.Reader) error {
	tarReader := tar.NewReader(source)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		finfo := hdr.FileInfo()

		switch {
		case finfo.IsDir():
			dir := path.Join(target, hdr.Name)
			if err = os.MkdirAll(dir, finfo.Mode()); err != nil {
				return err
			}
		case finfo.Mode().IsRegular():
			dest, err := os.Create(path.Join(target, hdr.Name))
			if err != nil {
				return err
			}
			if _, err = io.Copy(dest, tarReader); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported file type: %v", finfo.Mode().String())
		}
	}

	return nil
}
