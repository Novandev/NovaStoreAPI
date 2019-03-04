package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/joho/godotenv"
	"log"
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
	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()
accessKey := os.Getenv("ACCESS")
secretKey := os.Getenv("SECRET")
format := "\nAccess: %s\nSecret: %s\n"

_, authErr = fmt.Printf(format, accessKey, secretKey)
if authErr != nil {
log.Fatal(authErr.Error())
}


}