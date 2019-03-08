package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.mongodb.org/mongo-driver/mongo"
	"github.com/jinzhu/gorm"
	"github.com/joho/godotenv"
	"github.com/labstack/echo"
	"log"
	"net/http"
	"os"
)


type User struct {
	gorm.Model
	Username string
	Password string
}

//noinspection ALL
func main(){
authErr := godotenv.Load()
if authErr != nil {
log.Fatal("Error loading .env file")
}
	//db, err := gorm.Open("sqlite3", "test.db")
	//if err != nil {
	//	panic("failed to connect database")
	//}
	//defer db.Close()

accessKey := os.Getenv("ACCESS")
secretKey := os.Getenv("SECRET")
format := "\nAccess: %s\nSecret: %s\n"

_, authErr = fmt.Printf(format, accessKey, secretKey)
if authErr != nil {
log.Fatal(authErr.Error())
}

	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}
	e := echo.New()

// AWS Section


	// Open an AWS session in order to get access to buckets
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	uploader := s3manager.NewUploader(sess)



	//
	//// Echo fucntion sections
	//func getFile(c echo.Context) error {
	//
	//	// User ID from path `users/:id`
	//	userId := c.Param("id")
	//	// file name from
	//	file := c.Param("File")
	//
	//	f, err  := os.Open(file)
	//	if err != nil {
	//	return fmt.Errorf("failed to open file %q, %v", filename, err)
	//}
	//
	//	return c.String(http.StatusOK, f)
	//}
	//
	//
	//
	//
	//func getAllFiles(c echo.Context) error {
	//	// User ID from path `users/:id`
	//	userId := c.Param("id")
	//	return c.String(http.StatusOK, id)
	//}
	//
	//
	//
	//func saveFile(c echo.Context) error {
	//	// Get name
	//	userId := c.FormValue("userId")
	//	// Get avatar
	//	CSV, err := c.FormFile("CSV")
	//	if err != nil {
	//	return err
	//}
	//
	//	// Source
	//	src, err := avatar.Open()
	//	if err != nil {
	//	return err
	//}
	//	defer src.Close()
	//
	//	// Destination
	//	dst, err := os.Create(avatar.Filename)
	//	if err != nil {
	//	return err
	//}
	//	defer dst.Close()
	//
	//	// Copy
	//	if _, err = io.Copy(dst, src); err != nil {
	//	return err
	//}
	//
	//	return c.HTML(http.StatusOK, "<b>Thank you! " + name + "</b>")
	//}
	//
	//e.POST("user/:id/files", saveFile)
	//e.GET("user/:id/files/:id", getFile)
	//e.GET("user/:id/files/all", getAllFiles)
	//e.DELETE("user/:id/files/:id", deleteFile)
	//e.POST("/sign-up", newUser)
	//e.POST("/sign-in", signinUser)

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.Logger.Fatal(e.Start(":"+port))

}