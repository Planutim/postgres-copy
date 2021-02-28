package controllertests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/planutim/postgres-copy/api/models"
	"gopkg.in/go-playground/assert.v1"
)

func TestCreateUser(t *testing.T) {
	err := refreshUserTable()
	if err != nil {
		log.Fatal(err)
	}
	samples := []struct {
		inputJSON    string
		statusCode   int
		nickname     string
		email        string
		errorMessage string
	}{
		{
			inputJSON:    `{"nickname":"Pet", "email": "pet@gmail.com", "password": "password"}`,
			statusCode:   201,
			nickname:     "Pet",
			email:        "pet@gmail.com",
			errorMessage: "",
		},
		{
			inputJSON:    `{"nickname":"Frank", "email": "pet@gmail.com", "password": "password"}`,
			statusCode:   500,
			errorMessage: "Email Already Taken",
		},
		{
			inputJSON:    `{"nickname":"Pet", "email": "grand@gmail.com", "password": "password"}`,
			statusCode:   500,
			errorMessage: "Nickname Already Taken",
		},
		{
			inputJSON:    `{"nickname":"Kan", "email": "kangmail.com", "password": "password"}`,
			statusCode:   422,
			errorMessage: "Invalid Email",
		},
		{
			inputJSON:    `{"nickname": "", "email": "kan@gmail.com", "password": "password"}`,
			statusCode:   422,
			errorMessage: "Required Nickname",
		},
		{
			inputJSON:    `{"nickname": "Kan", "email": "", "password": "password"}`,
			statusCode:   422,
			errorMessage: "Required Email",
		},
		{
			inputJSON:    `{"nickname": "Kan", "email": "kan@gmail.com", "password": ""}`,
			statusCode:   422,
			errorMessage: "Required Password",
		},
	}

	for _, v := range samples {
		req, err := http.NewRequest("POST", "/users", bytes.NewBufferString(v.inputJSON))
		if err != nil {
			t.Errorf("this is the error: %v", err)
			return
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.CreateUser)
		handler.ServeHTTP(rr, req)

		responseMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
		if err != nil {
			fmt.Printf("Cannot convert to json: %v", err)
		}
		assert.Equal(t, rr.Code, v.statusCode)
		if v.statusCode == 201 {
			assert.Equal(t, responseMap["nickname"], v.nickname)
			assert.Equal(t, responseMap["email"], v.email)
		}
		if v.statusCode == 422 || v.statusCode == 500 && v.errorMessage != "" {
			assert.Equal(t, responseMap["error"], v.errorMessage)
		}
	}
}

func TestGetUsers(t *testing.T) {
	err := refreshUserTable()
	if err != nil {
		log.Fatal(err)
	}

	_, err = seedUsers()
	if err != nil {
		log.Fatal(err)
	}
	req, err := http.NewRequest("GET", "/users", nil)
	if err != nil {
		t.Errorf("this is the error: %v\n", err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetUsers)
	handler.ServeHTTP(rr, req)

	var users []models.User
	err = json.Unmarshal([]byte(rr.Body.String()), &users)
	if err != nil {
		log.Fatalf("Cannot convert to json: %v\n", err)
	}
	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Equal(t, len(users), 2)
}

func TestGetUserById(t *testing.T) {
	err := refreshUserTable()
	if err != nil {
		log.Fatal(err)
	}
	user, err := seedOneUser()
	if err != nil {
		log.Fatal(err)
	}

	userSample := []struct {
		id           string
		statusCode   int
		nickname     string
		email        string
		errorMessage string
	}{
		{
			id:         strconv.Itoa(int(user.ID)),
			statusCode: 200,
			nickname:   user.Nickname,
			email:      user.Email,
		}, {
			id:         "unkwnon",
			statusCode: 400,
		},
	}
	for _, v := range userSample {
		req, err := http.NewRequest("GET", "/users", nil)
		if err != nil {
			t.Errorf("This is the error: %v\n", err)
			continue
		}
		req = mux.SetURLVars(req, map[string]string{"id": v.id})
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.GetUser)
		handler.ServeHTTP(rr, req)

		responseMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
		if err != nil {
			t.Fatalf("Cannot convert to josn; %v\n", err)
		}
		assert.Equal(t, rr.Code, v.statusCode)
		if v.statusCode == 200 {
			assert.Equal(t, user.Nickname, responseMap["nickname"])
			assert.Equal(t, user.Email, responseMap["email"])
		}
	}
}

func TestUpdateUser(t *testing.T) {
	var AuthEmail, AuthPassword string
	var AuthID uint32

	err := refreshUserTable()
	if err != nil {
		log.Fatal(err)
	}

	users, err := seedUsers() // at least two users for test to work properly
	if err != nil {
		t.Fatalf("Error seeding user: %v\n", err)
	}
	// only suer 1
	for _, user := range users {
		if user.ID == 2 {
			continue
		}
		AuthID = user.ID
		AuthEmail = user.Email
		AuthPassword = "password"
	}
	token, err := server.SignIn(AuthEmail, AuthPassword)
	if err != nil {
		t.Fatalf("Cannot login: %v\n", err)
	}
	tokenString := fmt.Sprintf("Bearer %v", token)

	samples := []struct {
		id             string
		updateJSON     string
		statusCode     int
		updateNickname string
		updateEmail    string
		tokenGiven     string
		errorMessage   string
	}{
		{
			// ok
			id:             strconv.Itoa(int(AuthID)),
			updateJSON:     `{"nickname": "Grand", "email": "grand@gmail.com","password":"password"}`,
			statusCode:     200,
			updateNickname: "Grand",
			updateEmail:    "grand@gmail.com",
			tokenGiven:     tokenString,
			errorMessage:   "",
		}, {
			// no password
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"nickname":"Woman", "email": "woman@gmail.com","password": ""}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Password",
		},
		{
			// no token
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"nickname": "Man", "email": "man@gmail.com", "password":"password123"}`,
			statusCode:   401,
			tokenGiven:   "",
			errorMessage: "Unauthorized",
		}, {
			//when incorrect token was passed
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"nickname":"Woman", "email": "woman@gmail.com", "password":"woman123"}`,
			statusCode:   401,
			tokenGiven:   "wrong token",
			errorMessage: "Unauthorized",
		},
		{
			//remember kenny@gmail.com" belongs to user 2
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"nickname":"Frank", "email":"kenny@gmail.com","password": "password"}`,
			statusCode:   500,
			tokenGiven:   tokenString,
			errorMessage: "Email Already Taken",
		}, {
			//remember "Kenny Morris" belongs to user 2
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"nickname": "Kenny Morris", "email": "newemai@gmail.com", "password": "password123"}`,
			statusCode:   500,
			tokenGiven:   tokenString,
			errorMessage: "Nickname Already Taken",
		}, {
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"nickname": "Kan", "email": "wrongmail.com", "password": "password"}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Invalid Email",
		}, {
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"nickname": "Kan", "email": "", "password": "password"}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Email",
		}, {
			id:         "unknown",
			tokenGiven: tokenString,
			statusCode: 400,
		}, {
			//when user2 is using user1 token
			id:           strconv.Itoa(int(2)),
			updateJSON:   `{"nickname": "Mike", "email":"mike@gmail.com", "password": "mypassword"}`,
			statusCode:   401,
			tokenGiven:   tokenString,
			errorMessage: "Unauthorized",
		},
	}

	for _, v := range samples {
		req, err := http.NewRequest("POST", "/users", bytes.NewBufferString(v.updateJSON))
		if err != nil {
			t.Errorf("This is the rror: %v\n", err)
			continue
		}
		req = mux.SetURLVars(req, map[string]string{"id": v.id})

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.UpdateUser)
		req.Header.Set("Authorization", v.tokenGiven)
		handler.ServeHTTP(rr, req)

		responseMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
		if err != nil {
			t.Errorf("Cannot convert to json: %v\n", err)
			continue
		}
		assert.Equal(t, rr.Code, v.statusCode)
		if v.statusCode == 200 {
			assert.Equal(t, responseMap["nickname"], v.updateNickname)
			assert.Equal(t, responseMap["email"], v.updateEmail)
		} else if v.statusCode == 401 || v.statusCode == 422 || v.statusCode == 500 && v.errorMessage != "" {
			assert.Equal(t, v.errorMessage, responseMap["error"])
		}
	}
}

func TestDeleteUser(t *testing.T) {
	var AuthEmail, AuthPassword string
	var AuthID uint32

	err := refreshUserTable()
	if err != nil {
		log.Fatal(err)
	}

	users, err := seedUsers()

	if err != nil {
		log.Fatalf("Error seeding users; %v\n", err)
	}

	for _, user := range users {
		user.Password = "password"
		if user.ID == 2 {
			continue
		}

		AuthID = user.ID
		AuthEmail = user.Email
		AuthPassword = user.Password
	}
	token, err := server.SignIn(AuthEmail, AuthPassword)
	if err != nil {
		log.Fatalf("Cannot login: %v", err)
	}

	tokenString := fmt.Sprintf("Bearer %v", token)
	userSample := []struct {
		id           string
		tokenGiven   string
		statusCode   int
		errorMessage string
	}{
		{
			id:           strconv.Itoa(int(AuthID)),
			tokenGiven:   tokenString,
			statusCode:   204,
			errorMessage: "",
		}, {
			// no token
			id:           strconv.Itoa(int(AuthID)),
			tokenGiven:   "",
			statusCode:   401,
			errorMessage: "Unauthorized",
		}, {
			//wrong token
			id:           strconv.Itoa(int(AuthID)),
			tokenGiven:   "Wrong token",
			statusCode:   401,
			errorMessage: "Unauthorized",
		}, {
			id:         "unwkwon",
			tokenGiven: tokenString,
			statusCode: 400,
		}, {
			// user2 trying to use User1 token
			id:           "2",
			tokenGiven:   tokenString,
			statusCode:   401,
			errorMessage: "Unauthorized",
		},
	}
	for _, v := range userSample {
		req, err := http.NewRequest("DELETE", "/users", nil)
		if err != nil {
			t.Errorf("THis is the error: %v\n", err)
		}
		req = mux.SetURLVars(req, map[string]string{"id": v.id})

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.DeleteUser)

		req.Header.Set("Authorization", v.tokenGiven)

		handler.ServeHTTP(rr, req)
		assert.Equal(t, rr.Code, v.statusCode)

		if v.statusCode == 401 && v.errorMessage != "" {
			responseMap := make(map[string]interface{})
			err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
			if err != nil {
				t.Errorf("Cannot convert to json: %v\n", err)
			}
			assert.Equal(t, responseMap["error"], v.errorMessage)
		}
	}
}
