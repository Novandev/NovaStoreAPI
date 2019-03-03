package main
import (
"fmt"
"github.com/joho/godotenv"
"log"
"os"
)


func main(){
authErr := godotenv.Load()
if authErr != nil {
log.Fatal("Error loading .env file")
}
accessKey := os.Getenv("ACCESS")
secretKey := os.Getenv("SECRET")
format := "\nAccess: %s\nSecret: %s\n"

_, authErr = fmt.Printf(format, accessKey, secretKey)
if authErr != nil {
log.Fatal(authErr.Error())
}


}