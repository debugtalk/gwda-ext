package gwda_ext_opencv

import (
	"bytes"
	"errors"
	"github.com/electricbubble/gwda"
	cvHelper "github.com/electricbubble/opencv-helper"
	"image"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

// TemplateMatchMode is the type of the template matching operation.
type TemplateMatchMode int

const (
	// TmSqdiff maps to TM_SQDIFF
	TmSqdiff TemplateMatchMode = iota
	// TmSqdiffNormed maps to TM_SQDIFF_NORMED
	TmSqdiffNormed
	// TmCcorr maps to TM_CCORR
	TmCcorr
	// TmCcorrNormed maps to TM_CCORR_NORMED
	TmCcorrNormed
	// TmCcoeff maps to TM_CCOEFF
	TmCcoeff
	// TmCcoeffNormed maps to TM_CCOEFF_NORMED
	TmCcoeffNormed
)

type DebugMode int

const (
	// DmOff no output
	DmOff DebugMode = iota
	// DmEachMatch output matched and mismatched values
	DmEachMatch
	// DmNotMatch output only values that do not match
	DmNotMatch
)

type SessionExt struct {
	s               *gwda.Session
	scale           float64
	MatchMode       TemplateMatchMode
	Threshold       float64
	frame           *bytes.Buffer
	doneMjpegStream chan bool
}

// Extend 获得扩展后的 Session，
// 并指定匹配阀值，
// 获取当前设备的 Scale，
// 默认匹配模式为 TmCcoeffNormed，
// 默认关闭 OpenCV 匹配值计算后的输出
func Extend(session *gwda.Session, threshold float64, matchMode ...TemplateMatchMode) (sExt *SessionExt, err error) {
	sExt = &SessionExt{s: session}
	sExt.doneMjpegStream = make(chan bool, 1)

	if sExt.scale, err = sExt.s.Scale(); err != nil {
		return &SessionExt{}, err
	}

	if len(matchMode) == 0 {
		matchMode = []TemplateMatchMode{TmCcoeffNormed}
	}
	sExt.MatchMode = matchMode[0]
	cvHelper.Debug(cvHelper.DebugMode(DmOff))
	sExt.Threshold = threshold
	return sExt, nil
}

func (sExt *SessionExt) OnlyOnceThreshold(threshold float64) (newExt *SessionExt) {
	newExt = new(SessionExt)
	newExt.s = sExt.s
	newExt.scale = sExt.scale
	newExt.MatchMode = sExt.MatchMode
	newExt.Threshold = threshold
	return
}

func (sExt *SessionExt) OnlyOnceMatchMode(matchMode TemplateMatchMode) (newExt *SessionExt) {
	newExt = new(SessionExt)
	newExt.s = sExt.s
	newExt.scale = sExt.scale
	newExt.MatchMode = matchMode
	newExt.Threshold = sExt.Threshold
	return
}

func (sExt *SessionExt) Debug(dm DebugMode) {
	cvHelper.Debug(cvHelper.DebugMode(dm))
}

func (sExt *SessionExt) ConnectMjpegStream(httpClient *http.Client, mjpegURL string) (err error) {
	if httpClient == nil {
		return errors.New(`'httpClient' can't be nil`)
	}
	if mjpegURL == "" {
		return errors.New(`'mjpegURL' can't be an empty string`)
	}

	var req *http.Request
	if req, err = http.NewRequest(http.MethodGet, mjpegURL, nil); err != nil {
		return err
	}

	var resp *http.Response
	if resp, err = httpClient.Do(req); err != nil {
		return err
	}
	// defer func() { _ = resp.Body.Close() }()

	var boundary string
	if _, param, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err != nil {
		return err
	} else {
		boundary = strings.Trim(param["boundary"], "-")
	}

	mjpegReader := multipart.NewReader(resp.Body, boundary)

	go func() {
		for {
			select {
			case <-sExt.doneMjpegStream:
				_ = resp.Body.Close()
				return
			default:
				var part *multipart.Part
				if part, err = mjpegReader.NextPart(); err != nil {
					sExt.frame = nil
					continue
				}

				raw := new(bytes.Buffer)
				if _, err = raw.ReadFrom(part); err != nil {
					sExt.frame = nil
					continue
				}
				sExt.frame = raw
			}
		}
	}()

	return
}

func (sExt *SessionExt) CloseMjpegStream() {
	sExt.doneMjpegStream <- true
}

func (sExt *SessionExt) takeScreenshot() (raw *bytes.Buffer, err error) {
	if sExt.frame == nil {
		if raw, err = sExt.s.Screenshot(); err != nil {
			return nil, err
		}
	} else {
		raw = sExt.frame
	}
	return
}

// func (sExt *SessionExt) findImgRect(search string) (rect image.Rectangle, err error) {
// 	pathSource := filepath.Join(sExt.pathname, cvHelper.GenFilename())
// 	if err = sExt.s.ScreenshotToDisk(pathSource); err != nil {
// 		return image.Rectangle{}, err
// 	}
//
// 	if rect, err = cvHelper.FindImageRectFromDisk(pathSource, search, float32(sExt.Threshold), cvHelper.TemplateMatchMode(sExt.MatchMode)); err != nil {
// 		return image.Rectangle{}, err
// 	}
// 	return
// }

func (sExt *SessionExt) FindAllImageRect(search string) (rects []image.Rectangle, err error) {
	var bufSource, bufSearch *bytes.Buffer
	if bufSearch, err = getBufFromDisk(search); err != nil {
		return nil, err
	}
	if bufSource, err = sExt.takeScreenshot(); err != nil {
		return nil, err
	}

	if rects, err = cvHelper.FindAllImageRectsFromRaw(bufSource, bufSearch, float32(sExt.Threshold), cvHelper.TemplateMatchMode(sExt.MatchMode)); err != nil {
		return nil, err
	}
	return
}

func getBufFromDisk(name string) (*bytes.Buffer, error) {
	var f *os.File
	var err error
	if f, err = os.Open(name); err != nil {
		return nil, err
	}
	var all []byte
	if all, err = ioutil.ReadAll(f); err != nil {
		return nil, err
	}
	return bytes.NewBuffer(all), nil
}

func (sExt *SessionExt) FindImageRectInUIKit(search string) (x, y, width, height float64, err error) {
	var bufSource, bufSearch *bytes.Buffer
	if bufSearch, err = getBufFromDisk(search); err != nil {
		return 0, 0, 0, 0, err
	}
	if bufSource, err = sExt.takeScreenshot(); err != nil {
		return 0, 0, 0, 0, err
	}

	var rect image.Rectangle
	if rect, err = cvHelper.FindImageRectFromRaw(bufSource, bufSearch, float32(sExt.Threshold), cvHelper.TemplateMatchMode(sExt.MatchMode)); err != nil {
		return 0, 0, 0, 0, err
	}

	// if rect, err = sExt.findImgRect(search); err != nil {
	// 	return 0, 0, 0, 0, err
	// }
	x, y, width, height = sExt.MappingToRectInUIKit(rect)
	return
}

func (sExt *SessionExt) MappingToRectInUIKit(rect image.Rectangle) (x, y, width, height float64) {
	x, y = float64(rect.Min.X)/sExt.scale, float64(rect.Min.Y)/sExt.scale
	width, height = float64(rect.Dx())/sExt.scale, float64(rect.Dy())/sExt.scale
	return
}

func (sExt *SessionExt) PerformTouchActions(touchActions *gwda.WDATouchActions) error {
	return sExt.s.PerformTouchActions(touchActions)
}

func (sExt *SessionExt) PerformActions(actions *gwda.WDAActions) error {
	return sExt.s.PerformActions(actions)
}

// IsExist
