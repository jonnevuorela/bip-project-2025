package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gocv.io/x/gocv"
)

const (
	ModelPath = "models/traffic_accident_v112/weights/best.onnx"
	NamesPath = "models/traffic_accident_v112/classes.txt"
)

type AccidentModel struct {
	Net           gocv.Net
	ClassNames    []string
	OutputNames   []string
	InputWidth    int
	InputHeight   int
	InputChannels int
}

func LoadAccidentDetectionModel() *AccidentModel {
	model := &AccidentModel{
		InputWidth:    640,
		InputHeight:   640,
		InputChannels: 3,
	}

	var loadOk bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("RECOVER: Error loading model: %v\n", r)
				loadOk = false
			}
		}()
		model.Net = gocv.ReadNetFromONNX(ModelPath)
		loadOk = true
	}()

	if !loadOk {
		fmt.Println("Failed to load ONNX model!")
		return nil
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("RECOVER: Error configuring model: %v\n", r)
			}
		}()
		//	model.Net.SetPreferableBackend(gocv.NetBackendOpenCV)
		//	model.Net.SetPreferableTarget(gocv.NetTargetCPU)
	}()

	model.OutputNames = []string{"output0"}

	if _, err := os.Stat(ModelPath); os.IsNotExist(err) {
		fmt.Printf("ERROR: Model file not found at %s\n", ModelPath)
		return nil
	}

	classes, err := loadClassNames(NamesPath)
	if err != nil {
		fmt.Printf("Warning: Failed to load class names: %v\n", err)
		model.ClassNames = []string{"accident"}
	} else {
		model.ClassNames = classes
	}
	return model
}
func loadClassNames(path string) ([]string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("class names file not found: %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var classes []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		className := strings.TrimSpace(scanner.Text())
		if className != "" {
			classes = append(classes, className)
		}
	}

	return classes, scanner.Err()
}
