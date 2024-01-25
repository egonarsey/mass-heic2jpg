package main

import (
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"os"
	"path/filepath"
	s "strings"
	"time"

	"github.com/adrium/goheif"
)

type writerSkipper struct {
	w           io.Writer
	bytesToSkip int
}

func main() {
	oldHeicDir, err := getOldHeicDir(os.Args)
	fmt.Println("Process started. Working dir is set to ", oldHeicDir)
	if err != nil {
		log.Fatal(err)
	}
	heics, err := getHeicFilesInDir(oldHeicDir)
	if len(heics) != 0 {
		fmt.Println(len(heics), " heic files have been founded.")
		createNewHeicDir(oldHeicDir)
		createNewJpgDir(oldHeicDir)
	}
	i := 0
	for j, heic := range heics {
		oldHeicPath := filepath.Join(oldHeicDir, heic)
		newHeicPath := getNewHeicFilePath(oldHeicDir, heic)
		jpgPath := getJpgFilePath(oldHeicDir, heic)
		err := convertHeicToJpg(oldHeicPath, jpgPath)
		if err != nil {
			fmt.Printf("convertHeicToJpg() %q %q %q ", oldHeicPath, jpgPath, err.Error())
		} else {
			err := MoveFile(oldHeicPath, newHeicPath)
			if err != nil {
				fmt.Printf("MoveFile() %q %q %q ", oldHeicPath, newHeicPath, err.Error())
			}
		}
		i++
		if i == 100 {
			fmt.Println(time.Now(), " ", i, "files processed. ", len(heics)-j, " left")
			i = 0
		}
	}
}

func getHeicFilesInDir(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	files, err := f.ReadDir(0)
	if err != nil {
		return nil, err
	}
	var heicFiles []string
	for _, v := range files {
		if s.Contains(s.ToLower(v.Name()), ".heic") && !v.IsDir() {
			heicFiles = append(heicFiles, v.Name())
		}
	}
	return heicFiles, err
}

func convertHeicToJpg(input, output string) error {
	fileInput, err := os.Open(input)
	if err != nil {
		return err
	}
	defer fileInput.Close()

	exif, err := goheif.ExtractExif(fileInput)
	if err != nil {
		return err
	}

	img, err := goheif.Decode(fileInput)
	if err != nil {
		return err
	}

	fileOutput, err := os.OpenFile(output, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fileOutput.Close()

	w, _ := newWriterExif(fileOutput, exif)
	err = jpeg.Encode(w, img, nil)
	if err != nil {
		return err
	}
	return nil
}

func (w *writerSkipper) Write(data []byte) (int, error) {
	if w.bytesToSkip <= 0 {
		return w.w.Write(data)
	}

	if dataLen := len(data); dataLen <= w.bytesToSkip {
		w.bytesToSkip -= dataLen
		return dataLen, nil
	}

	if n, err := w.w.Write(data[w.bytesToSkip:]); err == nil {
		n += w.bytesToSkip
		w.bytesToSkip = 0
		return n, nil
	} else {
		return n, err
	}
}

func newWriterExif(w io.Writer, exif []byte) (io.Writer, error) {
	writer := &writerSkipper{w, 2}
	soi := []byte{0xff, 0xd8}
	if _, err := w.Write(soi); err != nil {
		return nil, err
	}

	if exif != nil {
		app1marker := 0xe1
		markerLen := 2 + len(exif)
		marker := []byte{0xff, uint8(app1marker), uint8(markerLen >> 8), uint8(markerLen & 0xff)}

		if _, err := w.Write(marker); err != nil {
			return nil, err
		}

		if _, err := w.Write(exif); err != nil {
			return nil, err
		}
	}
	return writer, nil
}

func MoveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("Couldn't open source file: %s", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("Couldn't open dest file: %s", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("Writing to output file failed: %s", err)
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("Failed removing original file: %s", err)
	}
	return nil
}

func createNewHeicDir(oldHeicDir string) error {
	d := filepath.Join(oldHeicDir, "heic")
	b, err := isDirExists(d)
	if b {
		fmt.Println("directory ", d, "is already exists")
		return nil
	}
	if err != nil {
		return err
	}
	return os.Mkdir(d, 0755)
}

func createNewJpgDir(oldHeicDir string) error {
	d := filepath.Join(oldHeicDir, "jpg")
	b, err := isDirExists(d)
	if b {
		fmt.Println("directory ", d, "is already exists")
		return nil
	}
	if err != nil {
		return err
	}
	return os.Mkdir(d, 0755)
}

func getNewHeicFilePath(oldHeicdir string, heicFileName string) string {
	return filepath.Join(oldHeicdir, "heic", heicFileName)
}

func getJpgFilePath(oldHeicdir string, heicFileName string) string {
	return filepath.Join(oldHeicdir, "jpg", s.ReplaceAll(s.ToLower(heicFileName), ".heic", ".jpg"))
}

// exists returns whether the given file or directory exists
func isDirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func getOldHeicDir(args []string) (string, error) {
	log.SetPrefix("mass-heic2jpeg: ")
	log.SetFlags(0)
	var oldHeicDir string
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	if len(args) == 0 || len(args) == 1 {
		oldHeicDir = wd
		log.Printf("No arguments given. Working dir is set to %q", oldHeicDir)
	} else {
		oldHeicDir = args[1]
		b, err := isDirExists(oldHeicDir)
		if err != nil {
			log.Fatal(err)
			return "", err
		}
		if !b {
			oldHeicDir = wd
			log.Printf("Directory does not exists. Working dir is set to %q", oldHeicDir)
		}
	}
	return oldHeicDir, err
}
