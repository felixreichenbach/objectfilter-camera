// This module implements a camera interface which takes a camera as a source and forwards the image to an object detector vision service.
// The vision service returns the detected labels and threshold, which are then filtered by this camera module based upon the confidence level configured.
// The boxes are then overlaid on the camera image and returned from this camera interface

package mycamera

import (
	"context"
	"fmt"
	"image"
	"slices"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/data"
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage/transform"
	"go.viam.com/rdk/services/vision"
	"go.viam.com/rdk/vision/objectdetection"
	"go.viam.com/utils"
)

func init() {
	resource.RegisterComponent(camera.API, Model, resource.Registration[camera.Camera, *Config]{Constructor: newObjectFilter})
}

var Model = resource.NewModel("felixreichenbach", "camera", "objectfilter")

// Maps JSON component configuration attributes.
type Config struct {
	// The camera image source
	Camera string
	// The list of vision services provided
	VisionServices []string `json:"vision_services"`
	// The labels to be filtered from the detection model. Default = none.
	Labels []string `json:"labels"`
	// Optional: The confidence threshold
	Confidence float64 `json:"confidence"`
	// Display bounding boxes or raw camera stream
	DisplayBoxes bool `json:"display_boxes"`
	// Activate/deactivate data recording filtering
	FilterData bool `json:"filter_data"`
}

// Configuration information validation, returning implicit dependencies.
func (cfg *Config) Validate(path string) ([]string, error) {
	if cfg.Camera == "" {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "camera")
	}
	if len(cfg.VisionServices) == 0 {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "vision_services")
	}
	impDeps := cfg.VisionServices
	impDeps = append(impDeps, cfg.Camera)
	return impDeps, nil
}

// The actual object filter camera
type objectFilter struct {
	resource.Named
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	conf   *Config
	logger logging.Logger

	camera         camera.Camera
	visionService  vision.Service
	visionServices map[string]vision.Service
}

// Constructor for the object filter camera
func newObjectFilter(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger) (camera.Camera, error) {
	newConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, err
	}
	of := &objectFilter{name: conf.ResourceName(), conf: newConf, logger: logger}
	of.camera, err = camera.FromDependencies(deps, newConf.Camera)
	if err != nil {
		return nil, err
	}
	of.visionServices = make(map[string]vision.Service)
	for _, visionService := range newConf.VisionServices {
		of.logger.Infof("VISION_SERVICE: %s", visionService)
		of.visionServices[visionService], err = vision.FromDependencies(deps, visionService)
		if err != nil {
			return nil, err
		}
	}
	of.visionService = of.visionServices[newConf.VisionServices[0]]
	return of, nil
}

// Returns the unfiltered source camera images
func (of *objectFilter) Images(ctx context.Context) ([]camera.NamedImage, resource.ResponseMetadata, error) {
	images, meta, err := of.camera.Images(ctx)
	if err != nil {
		return images, meta, err
	}
	return images, meta, nil
}

// Object filter camera does not implement PointClouds
func (*objectFilter) NextPointCloud(ctx context.Context) (pointcloud.PointCloud, error) {
	return nil, resource.ErrDoUnimplemented
}

// TODO: What does this API do?
func (of *objectFilter) Projector(ctx context.Context) (transform.Projector, error) {
	return of.camera.Projector(ctx)
}

// Returns the camera's supported properties
func (of *objectFilter) Properties(ctx context.Context) (camera.Properties, error) {
	p, err := of.camera.Properties(ctx)
	if err == nil {
		p.SupportsPCD = false
	}
	return p, err
}

// The camera image stream
func (of *objectFilter) Stream(ctx context.Context, errHandlers ...gostream.ErrorHandler) (gostream.VideoStream, error) {
	cameraStream, err := of.camera.Stream(ctx, errHandlers...)
	if err != nil {
		return nil, err
	}
	return filterStream{cameraStream, of}, nil
}

type filterStream struct {
	cameraStream gostream.VideoStream
	of           *objectFilter
}

// Gets the next image from the image stream
func (fs filterStream) Next(ctx context.Context) (image.Image, func(), error) {
	// Get next camera img
	img, release, err := fs.cameraStream.Next(ctx)
	if err != nil {
		return nil, nil, err
	}
	// Provide image to vision service and get object detections
	detections, err := fs.of.visionService.Detections(ctx, img, nil)
	if err != nil {
		return nil, nil, err
	}
	// Filter the detected labels according to the filter configuration
	var relevantdDetections []objectdetection.Detection
	for _, detection := range detections {
		if (slices.Contains(fs.of.conf.Labels, detection.Label())) && (detection.Score() >= fs.of.conf.Confidence) {
			relevantdDetections = append(relevantdDetections, detection)
		}
	}
	// In the case of a data manager request, no relevant detections and data filtering true return no capture
	if (ctx.Value(data.FromDMContextKey{}) == true) && (len(relevantdDetections) == 0) && fs.of.conf.FilterData {
		return nil, release, data.ErrNoCaptureToStore
	}
	// Overlay only the selected / configured detection labels and boxes onto the source image
	if (len(relevantdDetections) > 0) && fs.of.conf.DisplayBoxes {
		modImg, err := objectdetection.Overlay(img, relevantdDetections)
		if err != nil {
			return nil, release, fmt.Errorf("could not overlay bounding boxes: %w", err)
		}
		return modImg, release, nil
	}
	// return raw image
	return img, release, nil
}

// Closes the image stream
func (fs filterStream) Close(ctx context.Context) error {
	return fs.cameraStream.Close(ctx)
}

// DoCommand allows changing the vision service to be used dynamically
func (of *objectFilter) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	val, ok := cmd["vision-service"].(string)
	if ok {
		of.visionService = of.visionServices[val]
		return map[string]interface{}{"result": fmt.Sprintf("Vision service changed to: %s", val)}, nil
	}
	return nil, fmt.Errorf("vision service could not be changed to: %s", val)
}
