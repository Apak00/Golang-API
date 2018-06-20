package main

import (
    "fmt"
    "time"
    "net/smtp"

    "github.com/gin-gonic/gin"
    "github.com/jinzhu/gorm"
    _ "github.com/mattn/go-sqlite3"
    jwt_lib "github.com/dgrijalva/jwt-go"
    "github.com/gin-gonic/contrib/jwt"
)


type Users struct {
    Id        int    `gorm:"AUTO_INCREMENT" form:"id" json:"id"`
    Email string `gorm:"not null" form:"Email" json:"Email"`
    Password  string `gorm:"not null" form:"Password" json:"Password"`
    Username string `gorm:"not null" form:"Username" json:"Username"`
    Guid string `gorm:"not null" form:"Guid" json:"Guid"`
    Activated bool `gorm:"not null" form:"Activated" json:"Activated"`
    RegisterDate time.Time `gorm:"not null" form:"RegisterDate" json:"RegisterDate"`
}

func InitDb() *gorm.DB {
    // Openning file
    db, err := gorm.Open("sqlite3", "./data.db")
    // Display SQL queries
    db.LogMode(true)

    // Error
    if err != nil {
        panic(err)
    }
    // Creating the table
    if !db.HasTable(&Users{}) {
        db.CreateTable(&Users{})
        db.Set("gorm:table_options", "ENGINE=InnoDB").CreateTable(&Users{})
    }

    return db
}

func Cors() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Add("Access-Control-Allow-Origin", "http://localhost:4200")
        c.Next()
    }
}

func main() {
    r := gin.Default()

    r.Use(Cors())
    v1 := r.Group("api/v1")
    {
        v1.POST("/users", AddUser)
        v1.OPTIONS("/users", OptionsUser)

        v1.POST("/getuser", GetUser)
        v1.OPTIONS("/getuser", OptionsUser)

        v1.GET("/activateuser/:email", Activate_User)
        v1.OPTIONS("/activateuser/:email", OptionsUser)

        
        v1.PUT("/users/:id", UpdateUser)
        v1.DELETE("/users/:id", DeleteUser)
    }
    private := r.Group("/api/private")
    {
        private.GET("/users", GetUsers)
    }
    private.Use(jwt.Auth("mysupersecretpassword"))

    r.Run(":8080")

}

func GetUser(c *gin.Context) {

    var inc_user Users
    var user Users
    // Connection to the database
    db := InitDb()
    // Close connection database
    defer db.Close()


    

    c.Bind(&inc_user)

    // SELECT * FROM users WHERE email = c.Params.ByName("email");
    //db.First(&user, email)
    db.Raw("SELECT * FROM users WHERE users.email = ?", inc_user.Email).Scan(&user)

    if user.Email != "" && user.Password != "" && user.Password == inc_user.Password {
        if user.Activated == true{

            // Create the token
            token := jwt_lib.New(jwt_lib.GetSigningMethod("HS256"))
            // Set some claims
            token.Claims = jwt_lib.MapClaims{
                "Id":  user.Email + user.Username,
                "exp": time.Now().Add(time.Hour * 1).Unix(),

            }
            // Sign and get the complete encoded token as a string
            tokenString, err := token.SignedString([]byte("mysupersecretpassword"))
            if err != nil {
                c.JSON(500, gin.H{"message": "Could not generate token"})
            }
            c.JSON(200, gin.H{"token": tokenString})

        }else {
            c.JSON(405, gin.H{"error": "Please confirm your email"})
        }
    } else {
        // Display JSON error
        c.JSON(404, gin.H{"error": "User not found"})
    }

    // curl -i http://localhost:8080/api/v1/users/1
}

func AddUser(c *gin.Context) {
    var user Users
    var userExist Users
    db := InitDb()
    defer db.Close()

    c.Bind(&user)
    db.Raw("SELECT * FROM users WHERE users.email = ?", user.Email).Scan(&userExist)

    if(user.Email == userExist.Email){
        // Display error
        c.JSON(422, gin.H{"error": "Email is already exist!"})

    }else if user.Email != "" && user.Password != "" && user.Username != "" {
        
        user.RegisterDate = time.Now()
        // INSERT INTO "users" (name) VALUES (user.Name);
        
        if Send_Confirmation_Email(user) {
            db.Create(&user) 
            c.JSON(201, gin.H{"success": user})
        }else {
            c.JSON(400, gin.H{"error": "Email is not allowed"})
        }

        // Display error
        
    } else {
        // Display error
        c.JSON(422, gin.H{"error": "Fields are empty"})
    }

    // curl -i -X POST -H "Content-Type: application/json" -d "{ \"email\": \"Thea\", \"password\": \"Queen\" }" http://localhost:8080/api/v1/users
}


func Activate_User(c *gin.Context){
    var user Users
    db := InitDb()
    defer db.Close()

    email := c.Params.ByName("email")

    db.Raw("SELECT * FROM users WHERE users.email = ?", email).Scan(&user)
    timePassed := int((time.Now().Sub(user.RegisterDate)).Hours())
    if(timePassed >= 24 ){
        //Delete user if 24 hours passed between register and email confirmation
        user.Activated = false
        db.Delete(&user)
    }else{
        user.Activated = true
        // UPDATE users SET user WHERE id = user.Id;
        db.Save(&user)
        
    }

}


func Send_Confirmation_Email(user Users) bool{

    hostURL := "smtp.gmail.com"
    hostPort := "587"
    emailSender := "your email"
    password := "your email password"
    emailReciever := user.Email

    // Auth object
    emailAuth := smtp.PlainAuth(
        "",
        emailSender,
        password,
        hostURL,
        )

    msg := []byte("To: " + emailReciever + "\r\n" +
        "Subject: " + "Account Verification to Pragmalinq" + "\r\n" + "Please click the link below to activate your Account!\n\n" + 
        "http://localhost:8080/api/v1/activateuser/" + user.Email)

    err := smtp.SendMail(
        hostURL + ":" + hostPort,
        emailAuth,
        emailSender,
        []string{emailReciever},
        msg)
    if err != nil {
        fmt.Println("Error: ", err)
        return false
    }else{
        fmt.Println("Email Sent")
        return true
    }
    
}

func GetUsers(c *gin.Context) {
    // Connection to the database
    db := InitDb()
    // Close connection database
    defer db.Close()

    var users []Users
    // SELECT * FROM users
    db.Find(&users)

    // Display JSON result
    c.JSON(200, users)

    // curl -i http://localhost:8080/api/v1/users
}


func UpdateUser(c *gin.Context) {
    // Connection to the database
    db := InitDb()
    // Close connection database
    defer db.Close()

    // Get id user
    email := c.Params.ByName("email")
    var user Users
    // SELECT * FROM users WHERE id = 1;
    db.First(&user, email)

    if user.Email != "" && user.Password != "" {

        if user.Email != "" {
            var newUser Users
            c.Bind(&newUser)

            result := Users{
                Email: newUser.Email,
                Password:  newUser.Password,
            }

            // UPDATE users SET email='newUser.email', password='newUser.password' WHERE id = user.Id;
            db.Save(&result)
            // Display modified data in JSON message "success"
            c.JSON(200, gin.H{"success": result})
        } else {
            // Display JSON error
            c.JSON(404, gin.H{"error": "User not found"})
        }

    } else {
        // Display JSON error
        c.JSON(422, gin.H{"error": "Fields are empty"})
    }

    // curl -i -X PUT -H "Content-Type: application/json" -d "{ \"email\": \"Thea\", \"password\": \"Merlyn\" }" http://localhost:8080/api/v1/users/1
}

func DeleteUser(c *gin.Context) {
    // Connection to the database
    db := InitDb()
    // Close connection database
    defer db.Close()

    // Get id user
    email := c.Params.ByName("email")
    var user Users
    // SELECT * FROM users WHERE id = 1;
    db.First(&user, email)

    if user.Email != "" {
        // DELETE FROM users WHERE id = user.Id
        db.Delete(&user)
        // Display JSON result
        c.JSON(200, gin.H{"success": "User #" + email + " deleted"})
    } else {
        // Display JSON error
        c.JSON(404, gin.H{"error": "User not found"})
    }

    // curl -i -X DELETE http://localhost:8080/api/v1/users/1
}

func OptionsUser(c *gin.Context) {

    c.Writer.Header().Set("Access-Control-Allow-Methods", "DELETE,POST,PUT,GET")
    c.Writer.Header().Set("Access-Control-Allow-Headers",   "access-control-allow-headers,access-control-allow-origin,content-type")
    c.Next()
}
