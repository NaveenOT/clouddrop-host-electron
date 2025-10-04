package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var savePath string

// var userCollection *mongo.Collection
//var loggedIn bool
/*
type User struct {
	Username string `json:"username" bson:"username"`
	Password string `json:"password" bson:"password"`
}
*/
type meta struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Time string `json:"time"`
}

var secretKey = []byte("secretKey")

func main() {
	fmt.Println("Server V1.0")
	savePath = os.Args[1]
	port := os.Args[2]
	serverUsername := os.Args[3]
	//initMongoDB()
	initDB()
	r := gin.Default()
	//r.LoadHTMLGlob("templates/*")
	//r.GET("/", func(ctx *gin.Context) {
	//	ctx.HTML(http.StatusOK, "index.html", nil)
	//})
	//r.POST("/register", register)
	//r.POST("/login", login)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	auth := r.Group("/")
	auth.Use(authMiddleware(serverUsername))
	{
		auth.GET("/files", allfiles)
		auth.POST("upload", uploadfile)
		auth.GET("/download/:filename", downloadfile)
	}

	//r.GET("/files", allfiles)
	//r.GET("upload", uploadfile)
	//r.GET("/download/:filename", downloadfile)

	r.Run(":" + port)
}
func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./coulddrop.db")
	if err != nil {
		log.Fatal("Unable to Open Database Connection", err)
	}
	creation_query := `CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			type TEXT,
			time TEXT
		);`
	_, err = db.Exec(creation_query)
	if err != nil {
		log.Fatal("Error in Creation of Database", err)
	}

}

/*
	func initMongoDB() {
		client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
		if err != nil {
			log.Fatal("Mongodb Connection Error")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = client.Connect(ctx)
		if err != nil {
			log.Fatal(err)
		}
		userCollection = client.Database("clouddrop").Collection("users")
		fmt.Println("Connection Established")

}

	func register(c *gin.Context) {
		var user User
		err := c.ShouldBindJSON(&user)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid Request",
			})
			return
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error in Hashing Password",
			})
			return
		}
		user.Password = string(hashedPassword)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		exists, _ := userCollection.CountDocuments(ctx, bson.M{"username": user.Username})
		if exists > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Username already exists",
			})
			return
		}
		_, err = userCollection.InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Successfully Registered User",
		})
	}

	func login(c *gin.Context) {
		var user User
		err := c.ShouldBindJSON(&user)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid Request",
			})
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var storedUser User
		err = userCollection.FindOne(ctx, bson.M{"username": user.Username}).Decode(&storedUser)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username"})
			return
		}
		err = bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(user.Password))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Password"})
			return
		}
		//Creating a JWT only requires username
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": storedUser.Username,
			//"exp": time.Now().Add(time.Hour * 1).Unix(),
		})
		tokenString, err := token.SignedString(secretKey)
		c.JSON(http.StatusOK, gin.H{"message": "Login successful", "token": tokenString})
	}
*/
func authMiddleware(expectedUsername string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token unauthorized",
			})
			c.Abort()
			return
		}
		//Common thing in JWT
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}
		//verify the secret key
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return secretKey, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token Unauthorized or session expired",
			})
			c.Abort()
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid Token claims",
			})
			c.Abort()
			return
		}
		usernameClaim, ok := claims["username"].(string)
		if !ok || usernameClaim != expectedUsername {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Wrong Account",
			})
		}
		c.Set("username", usernameClaim)
		c.Next()
	}
}
func uploadfile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, "Error Reading File '%v'", err.Error())
		return
	}
	savePath := filepath.Join(savePath, filepath.Base(file.Filename))
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.String(http.StatusInternalServerError, "Error Saving file: '%v'", err.Error())
		return
	}
	_, err = db.Exec(`INSERT INTO files(name, type, time) VALUES(?, ?, ?);`,
		file.Filename, file.Header.Get("Content-Type"), time.Now().Format(time.RFC3339))
	if err != nil {
		c.String(http.StatusInternalServerError, "Error inserting into database '%v'", err.Error())
		return
	}
	c.String(http.StatusOK, "File '%s' Uploaded Successfully", file.Filename)
}

func downloadfile(c *gin.Context) {
	filename := c.Param("filename")
	path := filepath.Join(savePath, filepath.Base(filename))
	c.FileAttachment(path, filename)
}

func allfiles(c *gin.Context) {
	rows, err := db.Query(`SELECT name, type, time FROM files`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	defer rows.Close()
	var files []meta
	for rows.Next() {
		var f meta
		rows.Scan(&f.Name, &f.Type, &f.Time)
		files = append(files, f)
	}
	c.JSON(http.StatusOK, files)
}
