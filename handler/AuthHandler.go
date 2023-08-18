package handler

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/boj/redistore.v1"
	"gorm.io/gorm"
	. "gptbot/model"
	"log"
	"net/http"
	"os"
)

// https://github.com/gorilla/sessions
// https://github.com/karankumarshreds/GoAuthentication/blob/master/readme.md
var store, _ = redistore.NewRediStore(10, "tcp", ":6379", "", []byte(os.Getenv("SESSION_KEY")))

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

		userInfo, err := parseUsernamePassword(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		user := User{}
		// if no user found, user = User{}
		err = db.Limit(1).Find(&user, "username = ?", (*userInfo)["username"]).Error
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}
		if user == (User{}) {
			http.Error(w, "no such user, please register", http.StatusUnauthorized)
			return
		}

		// password is not correct
		if !checkPasswordHash((*userInfo)["password"], user.Password) {
			http.Error(w, "password is not correct", http.StatusUnauthorized)
			return
		}

		if err = createSession(w, r, user.Balance); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		if _, err = fmt.Fprint(w, "login successfully"); err != nil {
			log.Println(err)
			return
		}
	})
}

func LogoutHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get registers and returns a session for the given name and session store.
		session, _ := store.Get(r, "session_id")
		if session.IsNew {
			if _, err := fmt.Fprint(w, "you have not logged in yet"); err != nil {
				log.Println(err)
			}
			return
		}
		// if you want to delete session: https://github.com/gorilla/sessions/issues/160
		session.Values["authenticated"] = false
		if err := session.Save(r, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		if _, err := fmt.Fprint(w, "logout successfully"); err != nil {
			log.Println(err)
			return
		}
	})
}

func RegisterHandler(db *gorm.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method Not Supported", http.StatusMethodNotAllowed)
			return
		}

		userInfo, err := parseUsernamePassword(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		password, err := hashPassword((*userInfo)["password"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !db.Migrator().HasTable(&User{}) {
			// Migrate the schema - create table
			err = db.AutoMigrate(&User{})
			if err != nil {
				panic("failed to migrate the user schema")
			}
		}

		user := User{}
		err = db.Limit(1).Find(&user, "username = ?", (*userInfo)["username"]).Error
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		// user exists
		if user != (User{}) {
			http.Error(w, "username has been taken", http.StatusConflict)
			return
		}

		// user doesn't exist, process registration
		user = User{Username: (*userInfo)["username"], Password: password}
		if err = db.Create(&user).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
		}

		if _, err = fmt.Fprint(w, "registered successfully"); err != nil {
			log.Println(err)
			return
		}
	})
}

func createSession(w http.ResponseWriter, r *http.Request, tokens uint) error {
	session, err := store.New(r, "session_id")
	if err != nil {
		return fmt.Errorf("cannot create session: %w", err)
	}

	// MaxAge in seconds
	session.Options.MaxAge = 5 * 60
	session.Values["authenticated"] = true
	session.Values["tokens"] = tokens
	if err = session.Save(r, w); err != nil {
		return fmt.Errorf("cannot save session: %w", err)
	}

	return nil
}

func parseUsernamePassword(r *http.Request) (*map[string]string, error) {
	userInfo := make(map[string]string)
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("falied to parse form: %w", err)
	}

	userInfo["username"] = r.Form.Get("username")
	userInfo["password"] = r.Form.Get("password")
	if userInfo["username"] == "" || userInfo["password"] == "" {
		return nil, fmt.Errorf("no username or password")
	}

	return &userInfo, nil
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
