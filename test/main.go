package main

import (
	"io"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/", func(ctx *gin.Context) {
		f, err := os.Open("IMG_5675.jpg")
		if err != nil {
			log.Fatalln(err)
		}
		io.Copy(ctx.Writer, f)
		f.Close()
		ctx.Writer.Header().Set("content-type", "image/jpeg")
	})

	r.Run(":9000")
}
