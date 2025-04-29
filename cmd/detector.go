package main

import (
	"fmt"
	"strings"
	"time"

	"image"
	"image/color"
	"sort"

	"github.com/yalue/onnxruntime_go"
	"gocv.io/x/gocv"
)

type DetectionResult struct {
	ClassID    int
	ClassName  string
	Confidence float32
	BBox       BoundingBox
}

type BoundingBox struct {
	XMin float32
	YMin float32
	XMax float32
	YMax float32
}
type IndexedScore struct {
	index int
	score float32
}

const (
	ModelPath = "./models/yolo11n_mAP50-0697.onnx"
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
		fmt.Printf("\nmodel inputname: %v", inputs[i])
		inputNames = append(inputNames, inputs[i].Name)
	}
	var outputNames []string
	for i := range outputs {
		fmt.Printf("\nmodel outputname: %v", outputs[i])
		outputNames = append(outputNames, outputs[i].Name)
	}

	session, err := onnxruntime_go.NewSession(ModelPath, inputNames, outputNames, inputTensors, outputTensors)
	if err != nil {
		fmt.Printf("Error creating ONNX session: %v\n", err)
		return nil, nil, nil
	}

	fmt.Printf("\nonnxruntime version: %v", onnxruntime_go.GetVersion())
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

/**
 * Set of functions that take frames as parameter and
 * modify the input tensors data accordingly, then parses
 * the output tensors and lastly update UI (video feed and text section)
 * with results.
 * @param image.Image, *App
 * @return image.Image
 */
func processVideoFeed(img image.Image, app *App) image.Image {
	if app.Detector == nil || len(app.InputTensors) == 0 || len(app.OutputTensors) == 0 {
		app.DataLabel.SetText("Model not properly loaded")
		return img
	}

	startTime := time.Now()

	err := updateInputTensorWithImage(app.InputTensors[0], img)
	if err != nil {
		app.DataLabel.SetText(fmt.Sprintf("Error updating tensor: %v", err))
		return img
	}

	err = app.Detector.Run()
	if err != nil {
		app.DataLabel.SetText(fmt.Sprintf("Error running model: %v", err))
		return img
	}

	results := parseOutputTensor(app.OutputTensors[0])

	annotatedImg := drawDetectionResults(img, results)

	updateClassificationUI(app, results)

	elapsedMs := time.Since(startTime).Milliseconds()
	app.DataLabel.SetText(fmt.Sprintf("Inference time: %dms", elapsedMs))

	return annotatedImg
}

func updateInputTensorWithImage(tensor *onnxruntime_go.Tensor[float32], img image.Image) error {
	// size should match models training image size
	size := 640

	mat, err := gocv.ImageToMatRGBA(img)
	if err != nil {
		return fmt.Errorf("failed to convert image to Mat: %v", err)
	}
	defer mat.Close()

	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(mat, &resized, image.Point{X: size, Y: size}, 0, 0, gocv.InterpolationLinear)

	rgbMat := gocv.NewMat()
	defer rgbMat.Close()
	gocv.CvtColor(resized, &rgbMat, gocv.ColorBGRToRGB)

	tensorData := tensor.GetData()

	for c := 0; c < 3; c++ {
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				pixel := rgbMat.GetVecbAt(y, x)
				tensorData[c*size*size+y*size+x] = float32(pixel[c]) / 255.0
			}
		}
	}

	return nil
}
func parseOutputTensor(tensor *onnxruntime_go.Tensor[float32]) []DetectionResult {
	outputData := tensor.GetData()
	shape := tensor.GetShape()

	// adjustable thresholds for filtering detections
	const confThreshold = 0.25
	const iouThreshold = 0.45

	numDetections := int(shape[2])
	numValues := int(shape[1])

	var results []DetectionResult

	for i := 0; i < numDetections; i++ {
		x := outputData[0*numDetections+i]
		y := outputData[1*numDetections+i]
		w := outputData[2*numDetections+i]
		h := outputData[3*numDetections+i]

		classProbabilities := make([]float32, numValues-4)
		for c := 0; c < numValues-4; c++ {
			classProbabilities[c] = outputData[(c+4)*numDetections+i]
		}

		classID := 0
		maxProb := classProbabilities[0]
		for c := 1; c < len(classProbabilities); c++ {
			if classProbabilities[c] > maxProb {
				maxProb = classProbabilities[c]
				classID = c
			}
		}

		if maxProb < confThreshold {
			continue
		}

		xMin := x - w/2
		yMin := y - h/2
		xMax := x + w/2
		yMax := y + h/2

		results = append(results, DetectionResult{
			ClassID:    classID,
			ClassName:  getClassName(classID),
			Confidence: maxProb,
			BBox: BoundingBox{
				XMin: xMin,
				YMin: yMin,
				XMax: xMax,
				YMax: yMax,
			},
		})
	}

	results = applyNMS(results, iouThreshold)

	return results
}

func calculateIoU(box1, box2 BoundingBox) float32 {
	xMin := max(box1.XMin, box2.XMin)
	yMin := max(box1.YMin, box2.YMin)
	xMax := min(box1.XMax, box2.XMax)
	yMax := min(box1.YMax, box2.YMax)

	if xMax <= xMin || yMax <= yMin {
		return 0.0
	}

	intersectionArea := (xMax - xMin) * (yMax - yMin)

	box1Area := (box1.XMax - box1.XMin) * (box1.YMax - box1.YMin)
	box2Area := (box2.XMax - box2.XMin) * (box2.YMax - box2.YMin)
	unionArea := box1Area + box2Area - intersectionArea

	return intersectionArea / unionArea
}

func min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func applyNMS(detections []DetectionResult, iouThreshold float32) []DetectionResult {
	if len(detections) == 0 {
		return detections
	}

	sort.Slice(detections, func(i, j int) bool {
		return detections[i].Confidence > detections[j].Confidence
	})

	var result []DetectionResult

	result = append(result, detections[0])

	for i := 1; i < len(detections); i++ {
		keep := true

		for j := 0; j < len(result); j++ {
			if detections[i].ClassID == result[j].ClassID {
				iou := calculateIoU(detections[i].BBox, result[j].BBox)

				if iou > iouThreshold {
					keep = false
					break
				}
			}
		}

		if keep {
			result = append(result, detections[i])
		}
	}

	return result
}
func getClassName(classID int) string {
	labels := map[int]string{
		0: "accident",
		1: "vehicle",
	}

	if name, ok := labels[classID]; ok {
		return name
	}
	return fmt.Sprintf("Class %d", classID)
}

func drawDetectionResults(img image.Image, results []DetectionResult) image.Image {
	mat, err := gocv.ImageToMatRGBA(img)
	if err != nil {
		return img
	}
	defer mat.Close()

	colorMap := map[int]color.RGBA{
		0: {220, 0, 0, 220}, // Red for accident
		1: {0, 220, 0, 220}, // Green for vehicle
	}

	imgWidth := mat.Cols()
	imgHeight := mat.Rows()
	scaleX := float32(imgWidth) / 640.0
	scaleY := float32(imgHeight) / 640.0

	for _, res := range results {
		xMin := int(res.BBox.XMin * scaleX)
		yMin := int(res.BBox.YMin * scaleY)
		xMax := int(res.BBox.XMax * scaleX)
		yMax := int(res.BBox.YMax * scaleY)

		boxColor, ok := colorMap[res.ClassID]
		if !ok {
			boxColor = color.RGBA{0, 0, 255, 255} // Default blue
		}

		rect := image.Rect(xMin, yMin, xMax, yMax)
		gocv.Rectangle(&mat, rect, boxColor, 2)

		text := fmt.Sprintf("%s: %.1f%%", res.ClassName, res.Confidence*100)

		textSize := gocv.GetTextSize(text, gocv.FontHersheySimplex, 0.5, 1)
		gocv.Rectangle(&mat,
			image.Rect(xMin, yMin-textSize.Y-10, xMin+textSize.X, yMin),
			boxColor, -1)

		textPoint := image.Point{X: xMin, Y: yMin - 5}
		gocv.PutText(&mat, text, textPoint, gocv.FontHersheySimplex, 0.5, color.RGBA{255, 255, 255, 255}, 1)
	}

	imgDet, err := mat.ToImage()
	if err != nil {
		return img
	}
	return imgDet
}

func updateClassificationUI(app *App, results []DetectionResult) {
	var body strings.Builder

	classCounts := make(map[string]int)
	for _, res := range results {
		classCounts[res.ClassName]++
	}

	body.WriteString("Detection Summary:\n")
	body.WriteString("----------------\n")
	body.WriteString(fmt.Sprintf("Total detections: %d\n", len(results)))
	for class, count := range classCounts {
		body.WriteString(fmt.Sprintf("%s: %d\n", class, count))
	}
	body.WriteString("\n")

	maxResults := 5
	body.WriteString("Top Detections:\n")
	body.WriteString("----------------\n")
	for i, res := range results {
		if i >= maxResults {
			break
		}
		body.WriteString(fmt.Sprintf("%d. %s: %.1f%% [%.0f,%.0f,%.0f,%.0f]\n",
			i+1, res.ClassName, res.Confidence*100,
			res.BBox.XMin, res.BBox.YMin, res.BBox.XMax, res.BBox.YMax))
	}

	app.DataBody.SetText(body.String())
	app.DataBody.Refresh()
}
