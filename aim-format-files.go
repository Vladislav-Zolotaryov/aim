package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ValueOffset struct {
	start int
	end   int
}

type FailNamePair struct {
	original string
	new      string
}

func (e ValueOffset) ScanValue(data []byte) ([]byte, string) {
	rawSlice := data[e.start:e.end]
	firstEmpty := bytes.Index(rawSlice, []byte("\x00"))
	value := string(rawSlice[:firstEmpty])
	return rawSlice, value
}

const parseDateTimeFormat = "06-01-02 15:04:05"
const outputDateTimeFormat = "20060102_150405"

var (
	datetimeOffset       = ValueOffset{76, 94}
	motorcycleNameOffset = ValueOffset{1088, 1127}
	trackNameOffset      = ValueOffset{1128, 1167}
	riderNameOffset      = ValueOffset{1168, 1207}

	trackNameMapping = map[string]string{
		"Chaika":               "Chayka",
		"ChikaAuto":            "Chayka",
		"Ltava Moto":           "Ltava",
		"Chaikato":             "Chayka",
		"Chaykato":             "Chayka",
		"LtavaBaby":            "Ltava",
		"DniproKart-R":         "DniproKart R",
		"Chaikakarting":        "KartTochka",
		"Chaykakarting":        "KartTochka",
		"KartTochka RReverse)": "KartTochka R",
		"MotoParkro":           "MotorparkRomania",
	}
)

func updateFileTrackName(sourceDir string, destinationDir string, workingDirectory string, sourceFile string) FailNamePair {
	fileContents, err := os.ReadFile(filepath.Join(sourceDir, sourceFile))
	if err != nil {
		log.Fatal(err)
	}

	_, trackNameString := trackNameOffset.ScanValue(fileContents)
	_, riderNameString := riderNameOffset.ScanValue(fileContents)
	_, motorcycleNameString := motorcycleNameOffset.ScanValue(fileContents)
	_, dateTimeString := datetimeOffset.ScanValue(fileContents)

	parsedDate, err := time.Parse(parseDateTimeFormat, dateTimeString)
	if err != nil {
		log.Fatal(err)
	}
	dateTimeString = parsedDate.Format(outputDateTimeFormat)

	newFileName := strings.Join([]string{riderNameString, strings.ReplaceAll(motorcycleNameString, "+", ""), strings.ReplaceAll(trackNameString, " ", "-"), dateTimeString}, "_") + ".drk"

	// Maybe when I fiqure out how to reproduce the checksum
	// for key, value := range trackNameMapping {
	// 	if trackNameString == key {
	// 		copy(trackNameSlice, value)
	// 		fmt.Println("Replacing exact", trackNameString, "with", value, "in", filepath.Join(destinationDir, sourceFile))
	// 	} else if strings.Contains(trackNameString, key) {
	// 		newValue := strings.ReplaceAll(trackNameString, key, value)
	// 		copy(trackNameSlice, newValue)
	// 		fmt.Println("Replacing contains", trackNameString, "with", newValue, "in", filepath.Join(destinationDir, sourceFile))
	// 	}
	// }

	err = os.MkdirAll(filepath.Join(destinationDir, workingDirectory), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(destinationDir, workingDirectory, newFileName), fileContents, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	return FailNamePair{sourceFile, newFileName}
}

func main() {
	sourceDir := "./old"
	destinationDir := "./new"

	trackDirs, err := ioutil.ReadDir(sourceDir)
	if err != nil {
		log.Fatal(err)
	}

	fileNamePairs := []FailNamePair{}
	originalNonDrkFiles := []string{}

	for _, trackFile := range trackDirs {
		if trackFile.IsDir() {
			riderDirs, err := ioutil.ReadDir(trackFile.Name())
			if err != nil {
				log.Fatal(err)
			}
			for _, riderFile := range riderDirs {
				if riderFile.IsDir() {
					recordFiles, err := ioutil.ReadDir(filepath.Join(sourceDir, trackFile.Name(), riderFile.Name()))
					if err != nil {
						log.Fatal(err)
					}
					for _, recordFile := range recordFiles {
						if !recordFile.IsDir() && strings.HasSuffix(recordFile.Name(), ".drk") {
							fileNamePair := updateFileTrackName(sourceDir, destinationDir, filepath.Join(trackFile.Name(), riderFile.Name()), filepath.Join(trackFile.Name(), riderFile.Name(), recordFile.Name()))
							fileNamePairs = append(fileNamePairs, fileNamePair)
						} else {
							originalNonDrkFile := filepath.Join(trackFile.Name(), riderFile.Name(), recordFile.Name())
							originalNonDrkFiles = append(originalNonDrkFiles, originalNonDrkFile)
						}
					}
				}
			}
		}
	}

	for _, nonDrkFile := range originalNonDrkFiles {
		for _, fileNamePair := range fileNamePairs {
			withoutExtension := fileNamePair.original[:len(fileNamePair.original)-4]
			if strings.Contains(nonDrkFile, withoutExtension) {
				nonDrkExtension := nonDrkFile[len(fileNamePair.original)-4:]
				nonDrkFileName := nonDrkFile[strings.LastIndex(nonDrkFile, "/")+1 : len(fileNamePair.original)-4]
				nonDrkFilePath := nonDrkFile[:strings.LastIndex(nonDrkFile, "/")+1]

				newWithoutExtension := fileNamePair.new[strings.LastIndex(fileNamePair.new, "/")+1 : len(fileNamePair.new)-4]

				from := filepath.Join(sourceDir, nonDrkFilePath, nonDrkFileName+nonDrkExtension)
				to := filepath.Join(destinationDir, nonDrkFilePath, newWithoutExtension+nonDrkExtension)
				fmt.Println(from, "->", to)
				copy(from, to)
			}
		}
	}
}

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
