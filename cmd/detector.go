package main

import (
	"fmt"

	"github.com/yalue/onnxruntime_go"
)

const (
	ModelPath = "./models/best.onnx"
)

func LoadDetectionModel() (*onnxruntime_go.Session[float32], []*onnxruntime_go.Tensor[float32], []*onnxruntime_go.Tensor[float32]) {

	inputs, outputs, err := onnxruntime_go.GetInputOutputInfo(ModelPath)
	if err != nil {
		fmt.Printf("Error getting input/output info: %v", err)
	}
	inputTensors, errs := createTensors[float32](inputs)
	if errs != nil {
		for i := range errs {
			fmt.Printf("Errror creating inputTensor: %v", errs[i])
		}
	}
	outputTensors, errs := createTensors[float32](outputs)
	if errs != nil {
		for i := range errs {
			fmt.Printf("Error crateing outputTensor: %v", errs[i])
		}
	}
	var inputNames []string
	for i := range inputs {
		inputNames = append(inputNames, inputs[i].Name)
	}
	var outputNames []string
	for i := range outputs {
		outputNames = append(outputNames, outputs[i].Name)
	}

	session, err := onnxruntime_go.NewSession(ModelPath, inputNames, outputNames, inputTensors, outputTensors)
	if err != nil {
		fmt.Printf("Error creating ONNX session: %v\n", err)
		return nil, nil, nil
	}
	fmt.Printf("onnxruntime version: %v", onnxruntime_go.GetVersion())
	return session, inputTensors, outputTensors

}

func createTensors[T onnxruntime_go.TensorData](infos []onnxruntime_go.InputOutputInfo) ([]*onnxruntime_go.Tensor[T], []error) {
	var tensors []*onnxruntime_go.Tensor[T]
	var errs []error

	for i := range infos {
		elementCount := infos[i].Dimensions.FlattenedSize()

		data := make([]T, elementCount)

		tensor, err := onnxruntime_go.NewTensor(infos[i].Dimensions, data)
		if err != nil {
			errs = append(errs, fmt.Errorf("error creating tensor for input/output %d: %w", i, err))
			continue
		}

		tensors = append(tensors, tensor)
	}

	return tensors, errs
}
