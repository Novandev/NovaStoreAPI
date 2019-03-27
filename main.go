package main

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
	"log"
	"os"
	"time"
	//"strings"
)








//noinspection ALL
func main(){

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
// DB section
//mongoDbUname := os.Getenv("MLABSUSERNAME")
//mongoDbPassword := os.Getenv("MLABSPASSWORD")
//mongoDBUri := fmt.Sprintf("mongodb://novandev:Dad8e3cc@ds251197.mlab.com:51197/heroku_g6wjttm1",mongoDbUname,mongoDbPassword)
dbCtx, _ := context.WithTimeout(context.Background(), 100000*time.Second)
client, err := mongo.Connect(dbCtx, options.Client().ApplyURI(""))

UserCollection := client.Database("heroku_g6wjttm1").Collection("Users")

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
	//sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	//uploader := s3manager.NewUploader(sess)


	app := iris.Default()
	app.Logger().SetLevel("debug")
	// Recover from panics and log the panic message to the application's logger ("Warn" level).
	app.Use(recover.New())
	// logs HTTP requests to the application's logger ("Info" level)
	app.Use(logger.New())

	app.Get("/", func(context iris.Context) {
		context.WriteString("NovaStore")
	})
	app.Post("/register",func(ctx iris.Context) {
		var u User
		err := ctx.ReadJSON(&u)
		if err != nil {
			ctx.WriteString(err.Error())
			ctx.StatusCode(iris.StatusBadRequest)
			panic(err)
			return
		}
		// give the comment a unique ID and set the time
		u.ID = bson.NewObjectId()
		u.CreatedAt = time.Now()
		_, resErr := UserCollection.InsertOne(dbCtx,u)
		if resErr != nil {
			panic(resErr)
			ctx.WriteString(err.Error())
			ctx.StatusCode(iris.StatusBadRequest)

			return
		}
		//ctx.Application().Logger().Infof("received %#+v", u.Email)
		//ctx.Application().Logger().Infof("received %#+v", id)
		response := map[string]string{"status": "200", "UserID":u.ID.Hex() }
		ctx.JSON(response)
	})
	app.Post("/login",func(ctx iris.Context) {
		//var u User
		//err := ctx.ReadJSON(&u)
		//if err != nil {
		//	ctx.WriteString(err.Error())
		//	ctx.StatusCode(iris.StatusBadRequest)
		//	return
		//}
		//res, err := UserCollection.InsertOne(dbCtx, bson.M{"email": u.Email, "password": u.Password})
		//ctx.Application().Logger().Infof("received %#+v", u)
	})

	app.Run(iris.Addr(":"+port))
}

type (
	User struct {
		ID     bson.ObjectId `json:"id" bson:"_id,omitempty"`
		Email  string `json:"email"`
		Password string `json:"password"`
		CreatedAt time.Time `json:"CreatedAt"`
	}

)