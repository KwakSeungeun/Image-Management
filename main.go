package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"

	"github.com/disintegration/imaging"
)

const (
	maxWidth  = 1920
	maxHeight = 1080
	cropSizeUnit = 100
	brightUnit = 10	
	contrastUnit = 15
)

// DecodeImage decodes a single image by its name
func DecodeImage(filename string) (image.Image, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, _, err := image.Decode(f)
	if err != nil {
		fmt.Printf("Unable to decode %s", filename)
		return nil, err
	}
	return m, nil
}

func EncodeImage(filename string,src image.Image)(error){
	f, err := os.Create(filename)
	if err!=nil{
		return err
	}
	defer f.Close()
	jpeg.Encode(f, src, nil)
	return nil	
}

// ReadFiles recursively searches an entire directory for all the files in that directory
func ReadFiles(path string) []string {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	re := regexp.MustCompile("[.]")
	imgNames := []string{}
	for _, file := range files {
		fullPath := fmt.Sprintf("%s/%s", path, file.Name())
		if re.MatchString(file.Name()) {
			imgNames = append(imgNames, fullPath)
		} else {
			imgNames = append(imgNames, ReadFiles(fullPath)...)
		}
	}
	return imgNames
}

// DrawImage draw a single image on window
func DrawImage(
	ws *screen.Window,
	buffer *screen.Buffer,
	imgNames []string,
	index int) (error){
	src, err := DecodeImage(imgNames[index])
	if err != nil {
		return err;
	}
	source := (*buffer).RGBA()
	// draw background
	black := color.RGBA{0, 0, 0, 0}
	draw.Draw(source, (*buffer).Bounds(), &image.Uniform{black}, image.ZP, 1)
	// draw data image
	draw.Draw(source, src.Bounds(), src, image.ZP, 1)
	// upload image on screen
	(*ws).Upload(image.ZP, *buffer, (*buffer).Bounds())
	(*ws).Publish()
	return nil;
}

// DeleteFile deletes a single file path
func DeleteFile(imgNames []string, index int) ([]string, error) {
	err := os.Remove(imgNames[index])
	if err != nil {
		return nil, err
	}
	if len(imgNames) == 1 {
		imgNames = []string{}
		return imgNames, nil
	}
	switch index {
	case len(imgNames) - 1:
		// you have reached the end of the list
		imgNames = imgNames[:index]
	case 0:
		// you are at the start of the list
		imgNames = imgNames[1:]
	default:
		// you are somewhere between the end and the start of the list
		imgNames = append(imgNames[:index], imgNames[index+1:]...)
	}
	return imgNames, nil
}

// CheckOutOfIndex checks for index out of bounds errors
func CheckOutOfIndex(sliceLength int, index int) int {
	switch {
	case index >= sliceLength:
		return 0
	case index < 0:
		return sliceLength - 1
	default:
		return index
	}
}

func main() {
	var path string
	fmt.Println("Input path directory : ")
	fmt.Scanln(&path)

	driver.Main(func(s screen.Screen) {
		ws, err := s.NewWindow(nil)
		if err != nil {
			log.Fatal(fmt.Sprintf("Error creating a new window: %v", err))
		}
		defer ws.Release()

		buffer, err := s.NewBuffer(image.Pt(maxWidth, maxHeight))
		if err != nil {
			log.Fatal(fmt.Sprintf("Error creating a new buffer: %v", err))
		}
		defer buffer.Release()

		imgNames := ReadFiles(path)
		curIndex := 0
		err = DrawImage(&ws, &buffer, imgNames, curIndex)
		if(err!=nil){
			log.Fatal(err)
		}

		for {
			switch e := ws.NextEvent().(type) {
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}
			
			case key.Event:
				if e.Direction == key.DirRelease {
					switch e.Code {
					case key.CodeEscape:
						buffer.Release()
						return
					case key.CodeRightArrow:
						curIndex = CheckOutOfIndex(len(imgNames), curIndex+1)
						DrawImage(&ws, &buffer, imgNames, curIndex)
					case key.CodeLeftArrow:
						curIndex = CheckOutOfIndex(len(imgNames), curIndex-1)
						DrawImage(&ws, &buffer, imgNames, curIndex)
					case key.CodeDeleteForward, key.CodeDeleteBackspace:
						imgNames, err = DeleteFile(imgNames, curIndex)
						if err != nil {
							log.Fatal(fmt.Sprintf("Error deleteing a file : %v", err))
						}
						DrawImage(&ws, &buffer, imgNames, curIndex)
					
					
					
					
					case key.CodePageUp, key.CodePageDown, key.CodeDownArrow, key.CodeUpArrow:
						curImage,err := DecodeImage(imgNames[curIndex])
						if err!=nil{
							log.Fatal(err)
						}
						if e.Code == key.CodeUpArrow{
							curImage = imaging.AdjustBrightness(curImage, brightUnit)
						}else if e.Code == key.CodeDownArrow{
							curImage = imaging.AdjustBrightness(curImage, (-1)*brightUnit)
						}else if e.Code == key.CodePageUp{
							curImage = imaging.AdjustContrast(curImage, contrastUnit)
						}else if e.Code == key.CodePageDown{
							curImage = imaging.AdjustContrast(curImage, (-1)*contrastUnit)
						}
						
						err = EncodeImage(imgNames[curIndex], curImage)
						if err != nil{
							log.Fatal(fmt.Sprintf("Error encoding a file : %v", err))
						}
						DrawImage(&ws, &buffer, imgNames, curIndex)
					case key.CodeS :
						curImage, err := DecodeImage(imgNames[curIndex])
						if err != nil{
							log.Fatal(err)
						}
						width := curImage.Bounds().Max.X
						height := curImage.Bounds().Max.Y
						curImage = imaging.Crop(curImage,image.Rect(25,25,width-25,height-25))		
						err = EncodeImage(imgNames[curIndex],curImage)
						if err != nil{
							log.Fatal(fmt.Sprintf("Error encoding a file : %v", err))
						}			
						DrawImage(&ws, &buffer, imgNames, curIndex)
					}
				}
			}
		}
	})
}
