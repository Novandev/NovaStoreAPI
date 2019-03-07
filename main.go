package main

import (
	"fmt"
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
	e := echo.New()
	//e.POST("/files", saveFile)
	//e.GET("/files/:id", getFile)
	//e.DELETE("/files/:id", deleteFile)
	//e.POST("/sign-up", newUser)
	//e.POST("/sign-in", signinUser)

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.Logger.Fatal(e.Start(":"))

}