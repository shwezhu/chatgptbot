package handler

import (
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	. "gptbot/model"
	"net/http"
	"os"
)

/*type appError struct {
	Error   error
	Message string
	Code    int
}

type appHandler func(http.ResponseWriter, *http.Request) *appError

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}*/

// https://github.com/gorilla/sessions
// https://github.com/karankumarshreds/GoAuthentication/blob/master/readme.md
var cookieStore = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

// IndexHandler Function name starts with an uppercase letter: Public function
// And name starts with a lowercase letter: Private function
func IndexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Hi there")
		if err != nil {
			panic(err)
		}
	})
}

func LoginHandler(db *gorm.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method Not Supported", http.StatusMethodNotAllowed)
			return
		}

		username := ""
		password := ""
		if username, password = parseUsernamePassword(w, r); username == "" || password == "" {
			return
		}

		var userPtr *User
		var err error
		// no such user
		if userPtr, err = findUserByUserName(db, username); err != nil {
			http.Error(w, "username not found", http.StatusUnauthorized)
			return
		}

		// password is not correct
		if !checkPasswordHash(password, userPtr.Password) {
			http.Error(w, "password is not correct", http.StatusUnauthorized)
			return
		}

		// login successfully
		if session := createSession(w, r); session != nil {
			_, err = fmt.Fprint(w, "login successfully!")
			if err != nil {
				panic(err)
			}
		}
	})
}

func RegisterHandler(db *gorm.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method Not Supported", http.StatusMethodNotAllowed)
			return
		}

		username := ""
		password := ""
		var err error
		if username, password = parseUsernamePassword(w, r); username == "" || password == "" {
			return
		}

		if password, err = hashPassword(password); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user := User{Username: username, Password: password}
		if !db.Migrator().HasTable(&User{}) {
			// Migrate the schema - create table
			err := db.AutoMigrate(&User{})
			if err != nil {
				panic("failed to migrate the user schema")
			}
		}

		if _, err = findUserByUserName(db, username); err != nil {
			// user doesn't exist
			if errors.Is(err, gorm.ErrRecordNotFound) {
				err = db.Create(&user).Error
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					panic(err)
				}
				_, err = fmt.Fprint(w, "Registered successfully!")
				if err != nil {
					panic(err)
				}
				return
			} else { // other error
				http.Error(w, err.Error(), http.StatusInternalServerError)
				panic(err)
			}
		}

		// err == null is true, so the uer was found, exist
		http.Error(w, "Username has been taken", http.StatusConflict)
		return
	})
}

func createSession(w http.ResponseWriter, r *http.Request) *sessions.Session {
	// generate new session-id, if not exist
	if session, err := cookieStore.Get(r, "session-id"); err != nil {
		http.Error(w, fmt.Sprintf("Cannot create session: %v", err.Error()), http.StatusInternalServerError)
		panic(err.Error())
		return nil
	} else {
		// MaxAge in seconds
		session.Options.MaxAge = 5 * 60
		session.Values["authenticated"] = true
		if err = session.Save(r, w); err != nil {
			http.Error(w, fmt.Sprintf("Cannot create session: %v", err.Error()), http.StatusInternalServerError)
			panic(err.Error())
			return nil
		}
		return session
	}
}

func findUserByUserName(db *gorm.DB, username string) (*User, error) {
	user := User{}
	var err error
	// if user is found, set that object in &user
	// Avoid the ErrRecordNotFound error automatically printed on terminal,
	// so don't use db.First(&user, "username = ?", username).Error;
	// https://gorm.io/docs/query.html
	if err = db.Limit(1).Find(&user, "username = ?", username).Error; err == nil {
		return &user, nil
	}

	return nil, err
}

// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
// https://pkg.go.dev/golang.org/x/crypto/bcrypt
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func parseUsernamePassword(w http.ResponseWriter, r *http.Request) (string, string) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Please pass the data as URL form encoded", http.StatusBadRequest)
		return "", ""
	}

	username := r.Form.Get("username")
	password := r.Form.Get("password")
	if username == "" || password == "" {
		http.Error(w, "No username or password specified in the form", http.StatusBadRequest)
		return "", ""
	}

	return username, password
}
