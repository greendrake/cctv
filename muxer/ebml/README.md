# ebml-go

EBML (Extensible Binary Meta Language) is a binary and byte-aligned format that was originally developed for the Matroska audio-visual container. See https://matroska.org/ for details.

This Apache v2.0-licensed code comes from https://gitee.com/general252/ebml-go

Changes include:

    - removal of anything not necessarily necessary to write MKV files
    - removal of tagging of MKV files
    - removal of some debug logging
    - error handling changes
    - `go fmt`