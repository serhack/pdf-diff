package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"strconv"
)

// Structure for Pixel. Used as float to make operations more easily.
type Pixel struct {
	r,g,b,a float64
}

func rgbaToPixel(r uint32, g uint32, b uint32, a uint32) Pixel{
	return Pixel{float64(r), float64(g), float64(b), float64(a)}
}


func CreatePNG(PDFPath string) {

	fmt.Println("Image generation for: " + PDFPath)

	// Computes the sha256 hash
	folderName := ComputeSha256(PDFPath)

	// Checks if a folder with the name sha256(file) already exists
	if _, err := os.Stat(folderName); err == nil {
		return
	}

	// If not, probably we never met this pdf. Create the folder
	err := os.Mkdir(folderName, os.ModePerm)
	if err != nil {
		panic(err)
	}

	// Create the images
	cmd, _ := exec.Command("pdftoppm", "-png", PDFPath, folderName+"/png_gen").Output()
	fmt.Println(cmd)
}

func RetrievePixel(fileName string) ([][]Pixel, int, int) {
	infile, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer infile.Close()

	img, _, err := image.Decode(infile)
	if err != nil {
		panic(err)
	}

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y
	var pixels [][]Pixel
	for y := bounds.Min.Y; y < height; y++ {
		var row []Pixel
		for x := bounds.Min.X; x < width; x++ {
			row = append(row, rgbaToPixel(img.At(x, y).RGBA()))
		}
		pixels = append(pixels, row)
	}
	return pixels, width, height
}

func drawSection(row []Pixel) {
	for i := 0; i < len(row); i++ {
		row[i].g = row[i].g * 0.7
		row[i].b = row[i].b * 0.9
	}
}

func CompareSingleImage(path1 string, path2 string, i int) {

	sha1 := ComputeSha256(path1)
	sha2 := ComputeSha256(path2)

	// If the two images have the same hash, the two pages are the same.
	if sha1 == sha2{
		fmt.Printf("The pages number %d are the same.\n", i)
		return
	}

	pixel_1, x_1, y_1 := RetrievePixel(path1)
	pixel_2, x_2, y_2 := RetrievePixel(path2)

	if x_1 != x_2 {
		if y_1 != y_2 {
			fmt.Println("Warning: comparing two pdfs that do not have the same dimensions might cause some problems.")
		}
	}

	pixel_3 := pixel_2

	for y := 0; y < len(pixel_1); y++ {
		for x := 0; x < len(pixel_1[y]); x++ {
			result := compareSinglePixel(pixel_1[y][x], pixel_2[y][x])
			if !result {
				drawSection(pixel_3[y])
			}
		}
	}

	img := image.NewNRGBA(image.Rect(0, 0, x_1, y_1))
	for y := 0; y < y_1; y++ {
		for x := 0; x < x_1; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(pixel_3[y][x].r),
				G: uint8(pixel_3[y][x].g),
				B: uint8(pixel_3[y][x].b),
				A: uint8(pixel_3[y][x].a),
			})
		}
	}

	// Create the file under "generated" folder
	f, err := os.Create("generated/image-" + strconv.Itoa(i) + ".png")
	if err != nil {
		panic(err)
	}

	// Encode the image
	if err := png.Encode(f, img); err != nil {
		f.Close()
		panic(err)
	}

	if err := f.Close(); err != nil {
		panic(err)
	}

}

func compareSinglePixel(image1 Pixel, image2 Pixel) bool {
	// Returns true if two pixel are the same pixel
	if image1.b == image2.b && image1.g == image2.g && image1.r == image2.r && image1.a == image2.a {
		return true
	}
	return false
}

func ComputeSha256(filePath string) string {
	// Computes the hash of any file
	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func Compare(PDF1 string, PDF2 string) {
	// Compares the two files 

	shaPDF1 := ComputeSha256(PDF1)
	shaPDF2 := ComputeSha256(PDF2)

	err := os.Mkdir("generated", os.ModePerm)
	if err != nil {
		panic(err)
	}

	i := 1
	k := 1
	for {
		// pdftoppm creates pngs and the numbers are padded with a variable numbers of 0.
		// e.g. pdf contains <= 99 pages => 01.. 02.. 03..
		// pdf contains <= 999 pages => 001.. 002.. 003

		o := fmt.Sprintf("%d", k)
		s := fmt.Sprintf("%0" + o + "d", i)

		s_pdf1 := shaPDF1 + "/png_gen-" + s + ".png"
		s_pdf2 := shaPDF2 + "/png_gen-" + s + ".png"

		if _, err := os.Stat(s_pdf1); errors.Is(err, os.ErrNotExist) {
			// TODO: remove this println
			fmt.Println("File " + s_pdf1 + " does not exist.")
			k++
			if k == 12{
				break
			}
		} else {
			CompareSingleImage(s_pdf1, s_pdf2, i)
			i++
		}
		
	}

}

func main(){
	fmt.Println("pdf-diff: highlights the differences between two pdf files.")
	if len(os.Args) < 2 {
		fmt.Println("You need to specify two parameters!")
		os.Exit(1)
	}

	CreatePNG(os.Args[1])
	CreatePNG(os.Args[2])

	Compare(os.Args[1], os.Args[2])

}
