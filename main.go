package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/imroc/req"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
	uuid "github.com/nu7hatch/gouuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
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
	dbCtx, _ := context.WithTimeout(context.Background(), 100000*time.Second)
	client, err := mongo.Connect(dbCtx, options.Client().ApplyURI(mblabsUri))

	UserCollection := client.Database("novastoretest").Collection("Users")

	if err != nil {
		log.Fatal("Cannot connect to MLABS")
	}

	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	// AWS Section

	// Open an AWS session in order to get access to buckets
	bucket := os.Getenv("BUCKET") // Bucket for AWS access
	// sess, err := session.NewSession(&aws.Config{
	// 	Region: aws.String("us-east-1"),
	// })
	// if err != nil {
	// 	log.Fatal(err.Error)
	// }
	// uploader := s3manager.NewUploader(sess)

	app := iris.Default()
	app.Logger().SetLevel("debug")
	// Recover from panics and log the panic message to the application's logger ("Warn" level).
	app.Use(recover.New())
	// logs HTTP requests to the application's logger ("Info" level)
	app.Use(logger.New())

	app.Get("/", func(context iris.Context) {
		context.WriteString("NovaStore")
	})

	// ---------- REGISTER ROUTE ----------//
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
		u.HashUserPassword()
		u.GenerateAuthToken()
		u.NewId()
		_, resErr := UserCollection.InsertOne(dbCtx, u)
		if resErr != nil {
			panic(resErr)
			ctx.WriteString(err.Error())
			ctx.StatusCode(iris.StatusBadRequest)

			return
		}
		//ctx.Application().Logger().Infof("received %#+v", u.Email)
		//ctx.Application().Logger().Infof("received %#+v", id)
		response := map[string]string{"Status": "200", "Email Registered": u.Email, "Auth_token": u.Auth_token, "ID": u.ID}
		ctx.JSON(response)
	})

	// ---------- LOGIN ROUTE ----------//

	app.Post("/login", func(ctx iris.Context) {
		var u User
		err := ctx.ReadJSON(&u)
		if err != nil {
			ctx.WriteString(err.Error())
			ctx.StatusCode(iris.StatusBadRequest)
			return
		}

		res, err := UserCollection.Find(dbCtx, bson.M{"email": u.Email, "password": u.Password})
		fmt.Println(res)
		if err != nil {
			ctx.WriteString(err.Error())
			ctx.StatusCode(iris.StatusUnauthorized)
			return
		}
		response := map[string]string{"Status": "200", "Email": u.Email, "Auth_token": u.Auth_token, "ID": u.ID}
		ctx.JSON(response)
	})

	// ---------- Make Model ----------//
	app.Post("/make-model", func(ctx iris.Context) {
		file, info, err := ctx.FormFile("file")
		fmt.Println(info)
		target := ctx.URLParam("target")
		features := ctx.URLParam("features")
		model := ctx.URLParam("model")
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.HTML("Error while uploading: <b>" + err.Error() + "</b>")
			return
		}
		s := fmt.Sprintf("http://localhost:5000/make-model?target=%s&features=%s&model=%s", target, features, model)
		r, err := req.Post(s, req.FileUpload{
			File:      file,
			FieldName: "file", // FieldName is form field name
			FileName:  info.Filename,
		})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(r)
		// _, err = uploader.Upload(&s3manager.UploadInput{
		// 	Bucket: aws.String(bucket),
		// 	Key:    aws.String(info.Filename),
		// 	Body:   file,
		// })
		if err != nil {
			// Print the error and exit.
			fmt.Println("Unable to upload to bucket %q , %v", bucket, err)
			return
		}

		fmt.Printf("Successfully uploaded to %q\n", bucket)

		defer file.Close()
		// fmt.Println(info)
		// defer out.Close()
		return
	})

	app.Run(iris.Addr(":" + port))
}

// USER AND AUTHENTICATION SECTION.... TO BE MOVED Asap
var jwtKey = []byte(os.Getenv("JWT_SECRET"))

type (
	User struct {
		ID         string    `json:"id"`
		Email      string    `json:"email"`
		Password   string    `json:"password"`
		Auth_token string    `json:"auth_token"`
		Model      string    `json:"model"`
		Endpoint   string    `json:"endpoint"`
		CreatedAt  time.Time `json:"CreatedAt"`
	}
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (u *User) HashUserPassword() {
	password, _ := HashPassword(u.Password)
	u.Password = password
}

func (u *User) NewId() {
	id, _ := uuid.NewV4()
	u.ID = id.String()
}

// Helper string Claims for jwt
type Claims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

func (u *User) GenerateAuthToken() {

	claims := &Claims{
		Email:          u.Email,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign and get the complete encoded token as a string using the secret
	tokenString, _ := token.SignedString(jwtKey)

	u.Auth_token = tokenString
}
