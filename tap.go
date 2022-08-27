package uixt

import (
	"fmt"

	"github.com/electricbubble/gwda"
)

func (dExt *DriverExt) Tap(param interface{}) error {
	// click on coordinate: [x, y] should be relative
	if location, ok := param.([]float64); ok {
		// relative x,y of window size: [0.5, 0.5]
		if len(location) != 2 {
			return fmt.Errorf("invalid tap location params: %v", location)
		}
		x := location[0] * float64(dExt.windowSize.Width)
		y := location[1] * float64(dExt.windowSize.Height)
		return dExt.WebDriver.TapFloat(x, y)
	}

	// click on UI element
	if param, ok := param.(string); ok {
		return dExt.TapOffset(param, 0.5, 0.5)
	}

	return fmt.Errorf("invalid tap params: %v", param)
}

func (dExt *DriverExt) TapOffset(param string, xOffset, yOffset float64) (err error) {
	// click on element, find by name attribute
	ele, err := dExt.FindUIElement(param)
	if err == nil {
		return ele.Click()
	}

	var x, y, width, height float64
	if x, y, width, height, err = dExt.FindUIRectInUIKit(param); err != nil {
		return err
	}

	return dExt.WebDriver.TapFloat(x+width*xOffset, y+height*yOffset)
}

func (dExt *DriverExt) DoubleTap(param interface{}) (err error) {
	// click on coordinate: [x, y] should be relative
	if location, ok := param.([]float64); ok {
		// relative x,y of window size: [0.5, 0.5]
		if len(location) != 2 {
			return fmt.Errorf("invalid tap location params: %v", location)
		}
		x := location[0] * float64(dExt.windowSize.Width)
		y := location[1] * float64(dExt.windowSize.Height)
		return dExt.WebDriver.DoubleTapFloat(x, y)
	}

	// click on UI element
	if param, ok := param.(string); ok {
		return dExt.DoubleTapOffset(param, 0.5, 0.5)
	}

	return fmt.Errorf("invalid tap params: %v", param)
}

func (dExt *DriverExt) DoubleTapOffset(param string, xOffset, yOffset float64) (err error) {
	// click on element, find by name attribute
	ele, err := dExt.FindUIElement(param)
	if err == nil {
		return ele.DoubleTap()
	}

	var x, y, width, height float64
	if x, y, width, height, err = dExt.FindUIRectInUIKit(param); err != nil {
		return err
	}

	return dExt.WebDriver.DoubleTapFloat(x+width*xOffset, y+height*yOffset)
}

// TapWithNumber sends one or more taps
func (dExt *DriverExt) TapWithNumber(param string, numberOfTaps int) (err error) {
	return dExt.TapWithNumberOffset(param, numberOfTaps, 0.5, 0.5)
}

func (dExt *DriverExt) TapWithNumberOffset(param string, numberOfTaps int, xOffset, yOffset float64) (err error) {
	if numberOfTaps <= 0 {
		numberOfTaps = 1
	}
	var x, y, width, height float64
	if x, y, width, height, err = dExt.FindUIRectInUIKit(param); err != nil {
		return err
	}

	x = x + width*xOffset
	y = y + height*yOffset

	touchActions := gwda.NewTouchActions().Tap(gwda.NewTouchActionTap().WithXYFloat(x, y).WithCount(numberOfTaps))
	return dExt.PerformTouchActions(touchActions)
}
