# CCTV microservice
A purely Golang microservice to pull video from IP cameras, save it into MKV files and stream for on-demand view in web browsers.

Supports RTSP and DVRIP (Sofia) protocols.

MKV files are saved into 10-minute long chunks indo `<camera_name>/YYYY/MM/DD/HH-mm-ii.n.mkv` files.

Streams HEVC/H.265 video into web browsers that can play it (Chrome and some others). See `web-video-demo/index.html` for an example of frontend code to display these streams.

For configuration example see `config.yaml.example`.

Tested with TechAge and some BITVISION cameras.