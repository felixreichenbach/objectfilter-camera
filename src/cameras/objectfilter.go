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
	resource.RegisterComponent(camera.API, Model, resource.Registration[camera.Camera, *Config]{Constructor: newCamera})
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
type filterCamera struct {
	resource.Named
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	conf   *Config
	logger logging.Logger

	cam         camera.Camera
	vis         vision.Service
	visServices map[string]vision.Service
}

// Returns the unfiltered source camera images
func (fc *filterCamera) Images(ctx context.Context) ([]camera.NamedImage, resource.ResponseMetadata, error) {
	images, meta, err := fc.cam.Images(ctx)
	if err != nil {
		return images, meta, err
	}
	return images, meta, nil
}

// Object filter camera does not implement PointClouds
func (*filterCamera) NextPointCloud(ctx context.Context) (pointcloud.PointCloud, error) {
	return nil, resource.ErrDoUnimplemented
}

// TODO: What does this API do?
func (fc *filterCamera) Projector(ctx context.Context) (transform.Projector, error) {
	return fc.cam.Projector(ctx)
}

// Returns the camera's supported properties
func (fc *filterCamera) Properties(ctx context.Context) (camera.Properties, error) {
	p, err := fc.cam.Properties(ctx)
	if err == nil {
		p.SupportsPCD = false
	}
	return p, err
}

// The camera image stream
func (fc *filterCamera) Stream(ctx context.Context, errHandlers ...gostream.ErrorHandler) (gostream.VideoStream, error) {
	camStream, err := fc.cam.Stream(ctx, errHandlers...)
	if err != nil {
		return nil, err
	}
	return filterStream{camStream, fc}, nil
}

type filterStream struct {
	cameraStream gostream.VideoStream
	fc           *filterCamera
}

// Gets the next image from the image stream
func (fs filterStream) Next(ctx context.Context) (image.Image, func(), error) {
	image, release, err := fs.cameraStream.Next(ctx)
	if err != nil {
		return nil, nil, err
	}
	// Provide image to vision service and get object detections
	detections, err := fs.fc.vis.Detections(ctx, image, nil)
	if err != nil {
		return nil, nil, err
	}
	if len(detections) > 0 {
		var boxes []objectdetection.Detection
		for _, detection := range detections {
			if (slices.Contains(fs.fc.conf.Labels, detection.Label())) && (detection.Score() >= fs.fc.conf.Confidence) {
				boxes = append(boxes, detection)
			}
		}
		// Overlay only the selected / configured detection labels and boxes onto the source image
		modifiedImage, err := objectdetection.Overlay(image, boxes)
		if err != nil {
			return nil, nil, fmt.Errorf("could not overlay bounding boxes: %w", err)
		}
		return modifiedImage, release, nil
	}
	return image, release, nil
}

// Closes the image stream
func (fs filterStream) Close(ctx context.Context) error {
	return fs.cameraStream.Close(ctx)
}

// Constructor for the object filter camera
func newCamera(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger) (camera.Camera, error) {
	newConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, err
	}
	fc := &filterCamera{name: conf.ResourceName(), conf: newConf, logger: logger}
	fc.cam, err = camera.FromDependencies(deps, newConf.Camera)
	if err != nil {
		return nil, err
	}
	fc.visServices = make(map[string]vision.Service)
	for _, visionService := range newConf.VisionServices {
		fc.logger.Infof("VISION_SERVICE: %s", visionService)
		fc.visServices[visionService], err = vision.FromDependencies(deps, visionService)
		if err != nil {
			return nil, err
		}
	}
	fc.vis = fc.visServices[newConf.VisionServices[0]]
	return fc, nil
}

// DoCommand allows changing the vision service to be used dynamically
func (fc *filterCamera) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	val, ok := cmd["vision-service"].(string)
	if ok {
		fc.vis = fc.visServices[val]
		return map[string]interface{}{"result": fmt.Sprintf("Vision service changed to: %s", val)}, nil
	}
	return nil, fmt.Errorf("vision service could not be changed to: %s", val)
}
