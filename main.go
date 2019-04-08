package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
	//"strings"
)

//noinspection ALL
func main() {

	godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Error loading .env file")
	// }
	// DB section
	mblabsUri := os.Getenv("MONGOATLAS")
	bucket := os.Getenv("BUCKET")
	// fmt.Println(mblabsUri)
	//mongoDbUname := os.Getenv("MLABSUSERNAME")
	//mongoDbPassword := os.Getenv("MLABSPASSWORD")
	dbCtx, _ := context.WithTimeout(context.Background(), 100000*time.Second)
	client, err := mongo.Connect(dbCtx, options.Client().ApplyURI(mblabsUri))

	UserCollection := client.Database("novastoretest").Collection("Users")

	if err != nil {
		log.Fatal("Cannot connect to MLABS")
	}

	//accessKey := os.Getenv("ACCESS")
	//secretKey := os.Getenv("SECRET")

	//format := "\nAccess: %s\nSecret: %s\n"
	//_, authErr = fmt.Printf(format, accessKey, secretKey)
	//if authErr != nil {
	//log.Fatal(authErr.Error())
	//}

	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	// AWS Section

	// Open an AWS session in order to get access to buckets
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})
	if err != nil {
		log.Fatal(err.Error)
	}
	uploader := s3manager.NewUploader(sess)

	app := iris.Default()
	app.Logger().SetLevel("debug")
	// Recover from panics and log the panic message to the application's logger ("Warn" level).
	app.Use(recover.New())
	// logs HTTP requests to the application's logger ("Info" level)
	app.Use(logger.New())

	app.Get("/", func(context iris.Context) {
		context.WriteString("NovaStore")
	})
	app.Post("/register", func(ctx iris.Context) {
		var u User
		err := ctx.ReadJSON(&u)

		if err != nil {
			ctx.WriteString(err.Error())
			ctx.StatusCode(iris.StatusBadRequest)
			panic(err)
			return
		}
		// give the comment a unique ID and set the time
		u.CreatedAt = time.Now()
		_, resErr := UserCollection.InsertOne(dbCtx, u)
		if resErr != nil {
			panic(resErr)
			ctx.WriteString(err.Error())
			ctx.StatusCode(iris.StatusBadRequest)

			return
		}
		//ctx.Application().Logger().Infof("received %#+v", u.Email)
		//ctx.Application().Logger().Infof("received %#+v", id)
		response := map[string]string{"status": "200", "Email Registered": u.Email}
		ctx.JSON(response)
	})

	app.Post("/upload", func(ctx iris.Context) {
		fmt.Println("hit")
		file, _, err := ctx.FormFile("file")
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.HTML("Error while uploading: <b>" + err.Error() + "</b>")
			return
		}
		_, err = uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(bucket),
			Key:    aws.String("test"),
			Body:   file,
		})
		if err != nil {
			// Print the error and exit.
			fmt.Println("Unable to upload to bucket %q , %v", bucket, err)
		}

		fmt.Printf("Successfully uploaded to %q\n", bucket)

		defer file.Close()
		fmt.Println(reflect.TypeOf(file))
		// fmt.Println(info)
		// defer out.Close()
	})

	app.Post("/login", func(ctx iris.Context) {
		var u User
		err := ctx.ReadJSON(&u)
		if err != nil {
			ctx.WriteString(err.Error())
			ctx.StatusCode(iris.StatusBadRequest)
			return
		}
		res, err := UserCollection.Find(dbCtx, bson.M{"email": u.Email, "password": u.Password})
		if err != nil {
			ctx.WriteString(err.Error())
			ctx.StatusCode(iris.StatusBadRequest)
			return
		}
		fmt.Println(res)
	})

	app.Run(iris.Addr(":" + port))
}

type (
	User struct {
		ID        bson.ObjectId `json:"id" bson:"_id,omitempty"`
		Email     string        `json:"email"`
		Password  string        `json:"password"`
		CreatedAt time.Time     `json:"CreatedAt"`
	}
)
