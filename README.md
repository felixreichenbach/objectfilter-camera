# Viam Object Filter Camera

This camera module implements a Viam camera interface which filters objects returned by an object detection ML model based upon the label and confidence threshold you set.
You can find information regarding how to deploy Viam modules on to your smart machine in the [Viam documentation](https://docs.viam.com/registry/#use-modules)

You can also find the module in the Viam registry directly: [https://app.viam.com/module/felixreichenbach/object-filter](https://app.viam.com/module/felixreichenbach/object-filter)

## Component Configuration

```
    {
      "type": "camera",
      "namespace": "rdk",
      "attributes": {
        "labels": [
          "<-The Label(s) You Are Looking For->"
        ],
        "confidence": <-Your Confidence Threshold (e.g. 0.1)->
        "camera": "<-Your Camera Name->",
        "vision": "<-Your Vision Service->"
      },
      "depends_on": [
        "<-Your Camera Name->",
        "<-Your Vision Service->"
      ],
      "name": "filtercam",
      "model": "felixreichenbach:camera:objectfilter"
    }
```
