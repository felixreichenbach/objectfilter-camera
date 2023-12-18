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
        "vision_services": ["<-Your List OfVision Services->"]
      },
      "depends_on": [
        "<-Your Camera Name->",
        "<-Your Vision Service->"
      ],
      "name": "filtercam",
      "model": "felixreichenbach:camera:objectfilter"
    }
```

## Dynamically Choose Vision Service

The object filter camera supports changing the vision service applied to the source images dynamically via "do_command".
Simply provide the key/value pair as follows:

```
{"vision-service": "<-YOUR SERVICE NAME->}
```

The vision service must be contained in the list of the object filter component configuration "vision_services" attribute.
You can use "client.py" with your smart machine credentials to test it ```python client.py <service-name>```.
