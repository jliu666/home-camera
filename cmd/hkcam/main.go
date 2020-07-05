package main

import (
	"flag"
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"

	"image"
	"runtime"

	homecamera "github.com/jliu666/home-camera"
	"github.com/jliu666/home-camera/ffmpeg"
)

func main() {

	// Platform dependent flags
	var inputDevice *string
	var inputFilename *string
	var loopbackFilename *string
	var h264Encoder *string
	var h264Decoder *string

	if runtime.GOOS == "linux" {
		inputDevice = flag.String("input_device", "v4l2", "video input device")
		inputFilename = flag.String("input_filename", "/dev/video0", "video input device filename")
		loopbackFilename = flag.String("loopback_filename", "/dev/video1", "video loopback device filename")
		h264Decoder = flag.String("h264_decoder", "", "h264 video decoder")
		h264Encoder = flag.String("h264_encoder", "h264_omx", "h264 video encoder")
	} else if runtime.GOOS == "darwin" { // macOS
		inputDevice = flag.String("input_device", "avfoundation", "video input device")
		inputFilename = flag.String("input_filename", "default", "video input device filename")
		// loopback is not needed on macOS because avfoundation provides multi-access to the camera
		loopbackFilename = flag.String("loopback_filename", "", "video loopback device filename")
		h264Decoder = flag.String("h264_decoder", "", "h264 video decoder")
		h264Encoder = flag.String("h264_encoder", "libx264", "h264 video encoder")
	} else {
		log.Info.Fatalf("%s platform is not supported", runtime.GOOS)
	}

	var minVideoBitrate *int = flag.Int("min_video_bitrate", 0, "minimum video bit rate in kbps")
	var multiStream *bool = flag.Bool("multi_stream", false, "Allow mutliple clients to view the stream simultaneously")
	var dataDir *string = flag.String("data_dir", "Camera", "Path to data directory")
	var verbose *bool = flag.Bool("verbose", true, "Verbose logging")
	flag.Parse()

	if *verbose {
		log.Debug.Enable()
		ffmpeg.EnableVerboseLogging()
	}

	switchInfo := accessory.Info{Name: "Camera", FirmwareRevision: "0.0.9", Manufacturer: "Matthias Hochgatterer"}
	cam := accessory.NewCamera(switchInfo)

	cfg := ffmpeg.Config{
		InputDevice:      *inputDevice,
		InputFilename:    *inputFilename,
		LoopbackFilename: *loopbackFilename,
		H264Decoder:      *h264Decoder,
		H264Encoder:      *h264Encoder,
		MinVideoBitrate:  *minVideoBitrate,
		MultiStream:      *multiStream,
	}

	ffmpeg := homecamera.SetupFFMPEGStreaming(cam, cfg)

	t, err := hc.NewIPTransport(hc.Config{StoragePath: *dataDir}, cam.Accessory)
	if err != nil {
		log.Info.Panic(err)
	}

	t.CameraSnapshotReq = func(width, height uint) (*image.Image, error) {
		return ffmpeg.Snapshot(width, height)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
