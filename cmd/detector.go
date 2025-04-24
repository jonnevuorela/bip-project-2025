package main

import (
	"fmt"
	"image"
	"image/color"
	"sync"

	tf "github.com/galeone/tensorflow/tensorflow/go"
	op "github.com/galeone/tensorflow/tensorflow/go/op"
	"gocv.io/x/gocv"
)

var (
	yoloModel    *YoloModel
	modelInitMux sync.Mutex
)

type YoloModel struct {
	Graph   *tf.Graph
	Session *tf.Session
	Size    int64
	Classes int64
}

func InitializeYoloModel(weightsPath string) error {
	modelInitMux.Lock()
	defer modelInitMux.Unlock()

	if yoloModel != nil {
		return nil
	}

	scope := op.NewScope()
	size := int64(416)
	channels := int64(3)
	classes := int64(80)
	training := false

	fmt.Println("Initializing YOLO model...")
	graph, _ := YoloV3(scope, size, channels, yolo_anchors, yolo_anchor_masks, classes, training)

	session, err := tf.NewSession(graph, nil)
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}

	yoloModel = &YoloModel{
		Graph:   graph,
		Session: session,
		Size:    size,
		Classes: classes,
	}

	// Load weights (simplified for this example)
	fmt.Println("Loading weights from:", weightsPath)
	if err := load_darknet_weights(weightsPath); err != nil {
		return fmt.Errorf("failed to load weights: %v", err)
	}

	fmt.Println("YOLO model initialized successfully")
	return nil
}

// ProcessFrame processes a video frame with the YOLO model
func ProcessFrame(frame gocv.Mat) (gocv.Mat, []Detection) {
	if yoloModel == nil {
		return frame, nil
	}

	// Resize frame to match model input size
	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(frame, &resized, image.Pt(int(yoloModel.Size), int(yoloModel.Size)), 0, 0, gocv.InterpolationLinear)

	// Convert the frame to the format expected by the model
	// For a simplified version, we're just using the frame as is
	// In a real implementation, you would need to preprocess properly

	// For demonstration purposes, just draw a box on the frame
	result := frame.Clone()

	// Here you would:
	// 1. Convert the frame to a tensor
	// 2. Run the model session
	// 3. Process the results to get bounding boxes

	// Simulate some detections
	detections := []Detection{
		{
			Class:       "person",
			Confidence:  0.92,
			BoundingBox: image.Rect(100, 100, 300, 400),
		},
	}

	// Draw boxes
	for _, detection := range detections {
		DrawDetection(result, detection)
	}

	return result, detections
}

// Detection represents a detected object
type Detection struct {
	Class       string
	Confidence  float32
	BoundingBox image.Rectangle
}

// DrawDetection draws a bounding box and label for a detection
func DrawDetection(img gocv.Mat, detection Detection) {
	// Draw bounding box
	gocv.Rectangle(&img, detection.BoundingBox, color.RGBA{255, 0, 0, 255}, 2)

	// Create label text
	label := fmt.Sprintf("%s: %.2f", detection.Class, detection.Confidence)

	// Draw label background
	labelSize := gocv.GetTextSize(label, gocv.FontHersheySimplex, 0.5, 1)
	gocv.Rectangle(
		&img,
		image.Rect(
			detection.BoundingBox.Min.X,
			detection.BoundingBox.Min.Y-labelSize.Y-10,
			detection.BoundingBox.Min.X+labelSize.X,
			detection.BoundingBox.Min.Y,
		),
		color.RGBA{0, 0, 255, 255},
		-1,
	)

	// Draw label text
	gocv.PutText(
		&img,
		label,
		image.Point{detection.BoundingBox.Min.X, detection.BoundingBox.Min.Y - 5},
		gocv.FontHersheySimplex,
		0.5,
		color.RGBA{255, 255, 255, 255},
		1,
	)
}
