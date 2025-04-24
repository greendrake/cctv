package core

import "strings"

// TrackType https://www.matroska.org/technical/elements.html
// 1 - video, 2 - audio, 3 - complex, 16 - logo, 17 - subtitle, 18 - buttons, 32 - control, 33 - metadata
type TrackType = uint8

const (
	TrackTypeVideo    TrackType = 1
	TrackTypeAudio    TrackType = 2
	TrackTypeComplex  TrackType = 3
	TrackTypeLogo     TrackType = 16
	TrackTypeSubtitle TrackType = 17
	TrackTypeButtons  TrackType = 18
	TrackTypeControl  TrackType = 32
	TrackTypeMetadata TrackType = 33
)

// Official tags. See https://www.matroska.org/technical/tagging.html
const (
	TagAccompaniment         string = "ACCOMPANIMENT"
	TagActor                 string = "ACTOR"
	TagAddress               string = "ADDRESS"
	TagArranger              string = "ARRANGER"
	TagArtDirector           string = "ART_DIRECTOR"
	TagArtist                string = "ARTIST"
	TagAssistantDirector     string = "ASSISTANT_DIRECTOR"
	TagBPM                   string = "BPM"
	TagBPS                   string = "BPS"
	TagBarcode               string = "BARCODE"
	TagCatalogNumber         string = "CATALOG_NUMBER"
	TagCharacter             string = "CHARACTER"
	TagChoregrapher          string = "CHOREGRAPHER"
	TagComment               string = "COMMENT"
	TagComposer              string = "COMPOSER"
	TagComposerNationality   string = "COMPOSER_NATIONALITY"
	TagCompositionLocation   string = "COMPOSITION_LOCATION"
	TagConductor             string = "CONDUCTOR"
	TagContentType           string = "CONTENT_TYPE"
	TagCoproducer            string = "COPRODUCER"
	TagCopyright             string = "COPYRIGHT"
	TagCostumeDesigner       string = "COSTUME_DESIGNER"
	TagCountry               string = "COUNTRY"
	TagDateDigitized         string = "DATE_DIGITIZED"
	TagDateEncoded           string = "DATE_ENCODED"
	TagDatePurchased         string = "DATE_PURCHASED"
	TagDateRecorded          string = "DATE_RECORDED"
	TagDateReleased          string = "DATE_RELEASED"
	TagDateTagged            string = "DATE_TAGGED"
	TagDateWritten           string = "DATE_WRITTEN"
	TagDescription           string = "DESCRIPTION"
	TagDirector              string = "DIRECTOR"
	TagDirectorOfPhotography string = "DIRECTOR_OF_PHOTOGRAPHY"
	TagDistributedBy         string = "DISTRIBUTED_BY"
	TagEditedBy              string = "EDITED_BY"
	TagEmail                 string = "EMAIL"
	TagEncodedBy             string = "ENCODED_BY"
	TagEncoder               string = "ENCODER"
	TagEncoderSettings       string = "ENCODER_SETTINGS"
	TagExecutiveProducer     string = "EXECUTIVE_PRODUCER"
	TagFPS                   string = "FPS"
	TagFax                   string = "FAX"
	TagGenre                 string = "GENRE"
	TagIMDB                  string = "IMDB"
	TagISBN                  string = "ISBN"
	TagISRC                  string = "ISRC"
	TagInitialKey            string = "INITIAL_KEY"
	TagInstruments           string = "INSTRUMENTS"
	TagKeywords              string = "KEYWORDS"
	TagLCCN                  string = "LCCN"
	TagLabel                 string = "LABEL"
	TagLabelCode             string = "LABEL_CODE"
	TagLawRating             string = "LAW_RATING"
	TagLeadPerformer         string = "LEAD_PERFORMER"
	TagLicense               string = "LICENSE"
	TagLyricist              string = "LYRICIST"
	TagLyrics                string = "LYRICS"
	TagMCDI                  string = "MCDI"
	TagMasteredBy            string = "MASTERED_BY"
	TagMeasure               string = "MEASURE"
	TagMixedBy               string = "MIXED_BY"
	TagMood                  string = "MOOD"
	TagOriginal              string = "ORIGINAL"
	TagOriginalMediaType     string = "ORIGINAL_MEDIA_TYPE"
	TagPartNumber            string = "PART_NUMBER"
	TagPartOffset            string = "PART_OFFSET"
	TagPeriod                string = "PERIOD"
	TagPhone                 string = "PHONE"
	TagPlayCounter           string = "PLAY_COUNTER"
	TagProducer              string = "PRODUCER"
	TagProductionCopyright   string = "PRODUCTION_COPYRIGHT"
	TagProductionDesigner    string = "PRODUCTION_DESIGNER"
	TagProductionStudio      string = "PRODUCTION_STUDIO"
	TagPublisher             string = "PUBLISHER"
	TagPurchaseCurrency      string = "PURCHASE_CURRENCY"
	TagPurchaseInfo          string = "PURCHASE_INFO"
	TagPurchaseItem          string = "PURCHASE_ITEM"
	TagPurchaseOwner         string = "PURCHASE_OWNER"
	TagPurchasePrice         string = "PURCHASE_PRICE"
	TagRating                string = "RATING"
	TagRecordingLocation     string = "RECORDING_LOCATION"
	TagRemixedBy             string = "REMIXED_BY"
	TagReplayGainGain        string = "REPLAYGAIN_GAIN"
	TagReplayGainPeak        string = "REPLAYGAIN_PEAK"
	TagSample                string = "SAMPLE"
	TagScreenplayBy          string = "SCREENPLAY_BY"
	TagSortWith              string = "SORT_WITH"
	TagSoundEngineer         string = "SOUND_ENGINEER"
	TagSubject               string = "SUBJECT"
	TagSubtitle              string = "SUBTITLE"
	TagSummary               string = "SUMMARY"
	TagSynopsis              string = "SYNOPSIS"
	TagTMDB                  string = "TMDB"
	TagTVDB                  string = "TVDB"
	TagTermsOfUse            string = "TERMS_OF_USE"
	TagThanksTo              string = "THANKS_TO"
	TagTitle                 string = "TITLE"
	TagTotalParts            string = "TOTAL_PARTS"
	TagTuning                string = "TUNING"
	TagURL                   string = "URL"
	TagWrittenBy             string = "WRITTEN_BY"
)

const (
	CodecTypeVideo    = CodecType("Video")
	CodecTypeAudio    = CodecType("Audio")
	CodecTypeSubtitle = CodecType("Subtitle")
	CodecTypeButton   = CodecType("Button")
)

type CodecType = string

const (
	AudioCodecAAC            = "A_AAC"
	AudioCodecAAC2LC         = "A_AAC/MPEG2/LC"
	AudioCodecAAC2MAIN       = "A_AAC/MPEG2/MAIN"
	AudioCodecAAC2SBR        = "A_AAC/MPEG2/LC/SBR"
	AudioCodecAAC2SSR        = "A_AAC/MPEG2/SSR"
	AudioCodecAAC4LC         = "A_AAC/MPEG4/LC"
	AudioCodecAAC4LTP        = "A_AAC/MPEG4/LTP"
	AudioCodecAAC4MAIN       = "A_AAC/MPEG4/MAIN"
	AudioCodecAAC4SBR        = "A_AAC/MPEG4/LC/SBR"
	AudioCodecAAC4SSR        = "A_AAC/MPEG4/SSR"
	AudioCodecAC3            = "A_AC3"
	AudioCodecAC3BSID9       = "A_AC3/BSID9"
	AudioCodecAC3BSID10      = "A_AC3/BSID10"
	AudioCodecALAC           = "A_ALAC"
	AudioCodecATRACAT1       = "A_ATRAC/AT1"
	AudioCodecDTS            = "A_DTS"
	AudioCodecDTSEXPRESS     = "A_DTS/EXPRESS"
	AudioCodecDTSLOSSLESS    = "A_DTS/LOSSLESS"
	AudioCodecEAC3           = "A_EAC3"
	AudioCodecFLAC           = "A_FLAC"
	AudioCodecMLP            = "A_MLP"
	AudioCodecMPC            = "A_MPC"
	AudioCodecMP1            = "A_MPEG/L1"
	AudioCodecMP2            = "A_MPEG/L2"
	AudioCodecMP3            = "A_MPEG/L3"
	AudioCodecMSACM          = "A_MS/ACM"
	AudioCodecOPUS           = "A_OPUS"
	AudioCodecPCM            = "A_PCM/INT/LIT"
	AudioCodecPCMBE          = "A_PCM/INT/BIG"
	AudioCodecPCMFLOAT       = "A_PCM/FLOAT/IEEE"
	AudioCodecQUICKTIME      = "A_QUICKTIME"
	AudioCodecQUICKTIMEQDMC  = "A_QUICKTIME/QDMC"
	AudioCodecQUICKTIMEQDMC2 = "A_QUICKTIME/QDM2"
	AudioCodecREAL14         = "A_REAL/14_4"
	AudioCodecREAL28         = "A_REAL/28_8"
	AudioCodecREALCOOK       = "A_REAL/COOK"
	AudioCodecREALSIPR       = "A_REAL/SIPR"
	AudioCodecREALRALF       = "A_REAL/RALF"
	AudioCodecREALATRC       = "A_REAL/ATRC"
	AudioCodecTRUEHD         = "A_TRUEHD"
	AudioCodecTTA            = "A_TTA1"
	AudioCodecVORBIS         = "A_VORBIS"
	AudioCodecWAVPACK4       = "A_WAVPACK4"

	VideoCodecAV1          = "V_AV1"
	VideoCodecAVS2         = "V_AVS2"
	VideoCodecAVS3         = "V_AVS3"
	VideoCodecDIRAC        = "V_DIRAC"
	VideoCodecFFV1         = "V_FFV1"
	VideoCodecMPEG1        = "V_MPEG1"
	VideoCodecMPEG2        = "V_MPEG2"
	VideoCodecMPEG4ISOAP   = "V_MPEG4/ISO/AP"
	VideoCodecMPEG4ISOASP  = "V_MPEG4/ISO/ASP"
	VideoCodecMPEG4ISOAVC  = "V_MPEG4/ISO/AVC" // H264
	VideoCodecMPEG4ISOSP   = "V_MPEG4/ISO/SP"
	VideoCodecMPEG4MSV3    = "V_MPEG4/MS/V3"
	VideoCodecMPEGHISOHEVC = "V_MPEGH/ISO/HEVC" // H265
	VideoCodecMSCOMP       = "V_MS/VFW/FOURCC"
	VideoCodecPRORES       = "V_PRORES"
	VideoCodecQUICKTIME    = "V_QUICKTIME"
	VideoCodecREALV1       = "V_REAL/RV10"
	VideoCodecREALV2       = "V_REAL/RV20"
	VideoCodecREALV3       = "V_REAL/RV30"
	VideoCodecREALV4       = "V_REAL/RV40"
	VideoCodecTHEORA       = "V_THEORA"
	VideoCodecUNCOMPRESSED = "V_UNCOMPRESSED"
	VideoCodecVP8          = "V_VP8"
	VideoCodecVP9          = "V_VP9"

	SubtitleCodecDVBSUB     = "S_DVBSUB"
	SubtitleCodecHDMVPGS    = "S_HDMV/PGS"
	SubtitleCodecHDMVTEXTST = "S_HDMV/TEXTST"
	SubtitleCodecIMAGEBMP   = "S_IMAGE/BMP"
	SubtitleCodecKATE       = "S_KATE"
	SubtitleCodecTEXTASCII  = "S_TEXT/ASCII"
	SubtitleCodecTEXTASS    = "S_TEXT/ASS"
	// Deprecated: use SubtitleCodecTEXTASS instead
	SubtitleCodecASS     = "S_ASS"
	SubtitleCodecTEXTSSA = "S_TEXT/SSA"
	// Deprecated: use SubtitleCodecTEXTSSA instead
	SubtitleCodecSSA        = "S_SSA"
	SubtitleCodecTEXTUSF    = "S_TEXT/USF"
	SubtitleCodecTEXTUTF8   = "S_TEXT/UTF8"
	SubtitleCodecTEXTWEBVTT = "S_TEXT/WEBVTT"
	SubtitleCodecVOBSUB     = "S_VOBSUB"
	SubtitleCodecVOBSUBZLIB = "S_VOBSUB/ZLIB"

	ButtonCodecVOBBTN = "B_VOBBTN"
)

func CodecID(s string) (prefix CodecType, major, suffix string) {
	if len(s) < 2 && s[1] != '_' {
		panic("invalid codec id")
	}
	switch s[:2] {
	case "V_":
		prefix = CodecTypeVideo
	case "A_":
		prefix = CodecTypeAudio
	case "S_":
		prefix = CodecTypeSubtitle
	case "B_":
		prefix = CodecTypeButton
	}
	j := strings.Index(s, "/")
	if j == -1 {
		return prefix, s[2:], ""
	}
	return prefix, s[2:j], s[j+1:]
}
