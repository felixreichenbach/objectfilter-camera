// This module implements a camera interface which takes a camera as a source and forwards the image to an object detector vision service.
// The vision service returns the detected labels and threshold, which are then filtered by this camera module based upon the confidence level configured.
// The boxes are then overlaid on the camera image and returned from this camera interface

package mycamera

import (
	"context"
	"fmt"
	"image"

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
	// The raw camera source
	Camera string
	// List of vision services
	VisionServices []string `json:"vision_services"`
	// Optional: The labels to extracted. Default = none.
	Labels []string `json:"labels"`
	// Optional: The confidence threshold
	Confidence float64 `json:"confidence"`
}

// Implement component configuration validation and and return implicit dependencies.
func (cfg *Config) Validate(path string) ([]string, error) {

	if cfg.Camera == "" {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "camera")
	}

	if len(cfg.VisionServices) == 0 {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "vision_services")
	}

	impDeps := cfg.VisionServices
	impDeps = append(impDeps, cfg.Camera)
	//impDeps = append(impDeps, cfg.Vision)
	return impDeps, nil
	//return []string{cfg.Camera, cfg.Vision, cfg.VisionServices}, nil
}

// The object filter camera
type filterCamera struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	conf   *Config
	logger logging.Logger

	cam         camera.Camera
	vis         vision.Service
	visServices map[string]vision.Service
}

// Name implements camera.Camera.
func (fc *filterCamera) Name() resource.Name {
	return fc.name
}

// Images implements camera.Camera.
func (fc *filterCamera) Images(ctx context.Context) ([]camera.NamedImage, resource.ResponseMetadata, error) {
	images, meta, err := fc.cam.Images(ctx)
	if err != nil {
		return images, meta, err
	}

	return images, meta, nil
}

// NextPointCloud implements camera.Camera.
func (*filterCamera) NextPointCloud(ctx context.Context) (pointcloud.PointCloud, error) {
	return nil, resource.ErrDoUnimplemented
}

// Projector implements camera.Camera.
func (fc *filterCamera) Projector(ctx context.Context) (transform.Projector, error) {
	return fc.cam.Projector(ctx)
}

// Properties implements camera.Camera.
func (fc *filterCamera) Properties(ctx context.Context) (camera.Properties, error) {
	p, err := fc.cam.Properties(ctx)
	if err == nil {
		p.SupportsPCD = false
	}
	return p, err
}

// Stream implements camera.Camera.
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

// Close implements gostream.MediaStream.
func (fs filterStream) Close(ctx context.Context) error {
	return fs.cameraStream.Close(ctx)
}

// Next implements gostream.MediaStream.
func (fs filterStream) Next(ctx context.Context) (image.Image, func(), error) {

	image, release, err := fs.cameraStream.Next(ctx)
	if err != nil {
		return nil, nil, err
	}
	// Get detections from the image
	detections, err := fs.fc.vis.Detections(ctx, image, nil)
	if err != nil {
		return nil, nil, err
	}

	if len(detections) > 0 {
		var boxes []objectdetection.Detection
		for _, detection := range detections {
			if (contains(fs.fc.conf.Labels, detection.Label())) && (detection.Score() >= fs.fc.conf.Confidence) {
				// to be simplified with go version 1.21 which will introduce the slices package:
				//if (slices.Contains(fs.fc.conf.Labels, detection.Label())) && (detection.Score() >= fs.fc.conf.Confidence) {
				boxes = append(boxes, detection)
			}
		}
		// overlay detections of the source image
		modifiedImage, err := objectdetection.Overlay(image, boxes)
		if err != nil {
			return nil, nil, fmt.Errorf("could not overlay bounding boxes: %w", err)
		}
		return modifiedImage, release, nil
	}
	return image, release, nil
}

func contains(labels []string, label string) bool {
	for _, listlabel := range labels {
		if listlabel == label {
			return true
		}
	}
	return false
}

// Object filter camera constructor.
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

func (fc *filterCamera) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	val, ok := cmd["vision-service"].(string)
	if ok {
		fc.vis = fc.visServices[val]
		return map[string]interface{}{"result": fmt.Sprintf("Vision service changed to: %s", val)}, nil
	}
	return nil, fmt.Errorf("vision service could not be changed to: %s", val)
}
