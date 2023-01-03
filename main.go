package main

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"strconv"
)

// Structure for Pixel.
type Pixel struct {
	r, g, b, a uint8
	modified   bool
}

var rmaster, gmaster, bmaster float64
var hash1 string

func rgbaToPixel(r uint32, g uint32, b uint32, a uint32) Pixel {
	return Pixel{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8), false}
}

func CreatePNG(PDFPath string) {

	fmt.Println("Image generation for: " + PDFPath)

	// Computes the sha256 hash
	folderName := ComputeSha256(PDFPath)

	// Checks if a folder with the name sha256(file) already exists
	if _, err := os.Stat("data/" + folderName); err == nil {
		return
	}

	// If not, probably we never met this pdf. Create the folder
	err := os.Mkdir("data/" +folderName, os.ModePerm)
	if err != nil {
		panic(err)
	}

	file, err := os.Create("data/" + folderName + "/.tmp")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Create the images
	cmd, _ := exec.Command("pdftoppm", "-png", PDFPath, "data/" + folderName+"/png_gen").Output()
	fmt.Println(cmd)

	err = os.Remove("data/" + folderName + ".tmp")
	if err != nil {
		panic(err)
	}
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
	pixels := make([][]Pixel, bounds.Max.Y)
	for y := bounds.Min.Y; y < height; y++ {
		row := make([]Pixel, bounds.Max.X)
		for x := bounds.Min.X; x < width; x++ {
			row[x] = rgbaToPixel(img.At(x, y).RGBA())
		}
		pixels[y] = row
	}
	return pixels, width, height
}

func drawSection(row []Pixel) {
	alpha := 0.6
	notalpha := float64(1 - alpha)

	for i := 0; i < len(row); i++ {

		if !row[i].modified {
			row[i].r = uint8(float64(row[i].r)*alpha + notalpha*rmaster)
			row[i].g = uint8(float64(row[i].g)*alpha + notalpha*gmaster)
			row[i].b = uint8(float64(row[i].b)*alpha + notalpha*bmaster)
			row[i].modified = true
		}

	}
}

func CompareSingleImage(path1 string, path2 string, i int) {

	sha1 := ComputeSha256(path1)
	sha2 := ComputeSha256(path2)

	// If the two images have the same hash, the two pages are the same.
	if sha1 == sha2 {
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
			if !pixel_3[y][x].modified {
				result := compareSinglePixel(pixel_1[y][x], pixel_2[y][x])
				if !result {
					drawSection(pixel_3[y])
				}
			}
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, x_1, y_1))
	for y := 0; y < y_1; y++ {
		for x := 0; x < x_1; x++ {
			img.Set(x, y, color.RGBA{
				R: pixel_3[y][x].r,
				G: pixel_3[y][x].g,
				B: pixel_3[y][x].b,
				A: pixel_3[y][x].a,
			})
		}
	}

	// Create the file under "generated" folder
	f, err := os.Create("data/generated/" + hash1 + "/image-" + strconv.Itoa(i) + ".png")
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
	hash1 = shaPDF1
	shaPDF2 := ComputeSha256(PDF2)

	if _, err := os.Stat("data"); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir("data", os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	if _, err := os.Stat("data/generated"); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir("data/generated", os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	if _, err := os.Stat("data/generated/" + shaPDF1); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir("data/generated/" + shaPDF1, os.ModePerm)
		if err != nil {
			panic(err)
		}
	} else {
		return
	}

	i := 1
	k := 1
	for {
		// pdftoppm creates pngs and the numbers are padded with a variable numbers of 0.
		// e.g. pdf contains <= 99 pages => 01.. 02.. 03..
		// pdf contains <= 999 pages => 001.. 002.. 003

		o := fmt.Sprintf("%d", k)
		s := fmt.Sprintf("%0"+o+"d", i)

		s_pdf1 := "data/" +shaPDF1 + "/png_gen-" + s + ".png"
		s_pdf2 := "data/" + shaPDF2 + "/png_gen-" + s + ".png"

		if _, err := os.Stat(s_pdf1); errors.Is(err, os.ErrNotExist) {
			k++
			if k == 12 {
				break
			}
		} else {
			CompareSingleImage(s_pdf1, s_pdf2, i)
			i++
		}

	}
}

func hexToRGB(hexcolor string) {
	// converts a string to rgb values
	values, _ := strconv.ParseUint(hexcolor, 16, 32)
	rmaster = float64(values >> 16)
	gmaster = float64((values >> 8) & 0xff)
	bmaster = float64((values) & 0xff)

	fmt.Printf("Color chosen: %f %f %f \n", rmaster, gmaster, bmaster)

}

func main() {

	// flags

	color := flag.String("color", "ff2010", "hex value for the background color for highlighting")
	enableServer := flag.Bool("server", false, "flag to enable local server for pdf-diff")

	flag.Parse()

	if !*enableServer {
		arguments := flag.Args()

		if len(arguments) < 2 {
			fmt.Println("pdf-diff: highlights the differences between two pdf files.")
			fmt.Println("Usage: pdf-diff pdf-file-1 pdf-file-2 [-color] hex-color")
			fmt.Println()
			flag.PrintDefaults()
			os.Exit(1)
		}

		hexToRGB(*color)
		CreatePNG(arguments[0])
		CreatePNG(arguments[1])
		Compare(arguments[0], arguments[1])
	} else {
		StartServer()
	}

}
