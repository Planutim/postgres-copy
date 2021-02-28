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

func TestCreatePost(t *testing.T) {
	err := refreshUserAndPostTable()
	if err != nil {
		log.Fatal(err)
	}

	user, err := seedOneUser()
	user.Password = "password"
	if err != nil {
		log.Fatalf("Cannot seed user %v\n", err)
	}

	token, err := server.SignIn(user.Email, user.Password)
	if err != nil {
		log.Fatalf("Cannot login: %v\n", err)
	}
	tokenString := fmt.Sprintf("Bearer %v", token)

	samples := []struct {
		inputJSON    string
		statusCode   int
		title        string
		content      string
		author_id    uint32
		tokenGiven   string
		errorMessage string
	}{
		{
			inputJSON:    `{"title": "The title", "content": "the content","author_id": 1}`,
			statusCode:   201,
			tokenGiven:   tokenString,
			title:        "The title",
			content:      "the content",
			author_id:    user.ID,
			errorMessage: "",
		}, {
			inputJSON:    `{"title": "The title", "content":"the content", "author_id":1}`,
			statusCode:   500,
			tokenGiven:   tokenString,
			errorMessage: "Title Already Taken",
		}, {
			// no token
			inputJSON:    `{"title": "When no token is passed", "content": "the content", "author_id": 1}`,
			statusCode:   401,
			tokenGiven:   "",
			errorMessage: "Unauthorized",
		}, {
			// incorrect token
			inputJSON:    `{"title": "When incorrect token is passed", "content": "The content", "author_id": 1}`,
			statusCode:   401,
			tokenGiven:   "Wrong token",
			errorMessage: "Unauthorized",
		}, {
			// no title
			inputJSON:    `{"title": "","content": "this is content", "author_id": 1}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Title",
		}, {
			inputJSON:    `{"title": "This is a title", "content": "", "author_id": 1}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Content",
		}, {
			inputJSON:    `{"title": "This is thhe awesome title!", "content": "Another content"}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Author",
		}, {
			// when user2 attempts to use user1 token
			inputJSON:    `{"title":"This is the title", "content": "the content", "author_id": 2}`,
			statusCode:   401,
			tokenGiven:   tokenString,
			errorMessage: "Unauthorized",
		},
	}
	for _, v := range samples {
		req, err := http.NewRequest("POST", "/posts", bytes.NewBufferString(v.inputJSON))
		if err != nil {
			t.Errorf("this is the error: %v\n", err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.CreatePost)

		req.Header.Set("Authorization", v.tokenGiven)
		handler.ServeHTTP(rr, req)

		responseMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
		if err != nil {
			fmt.Printf("Cannot convert to json: %v", err)
		}

		assert.Equal(t, rr.Code, v.statusCode)

		if v.statusCode == 201 {
			assert.Equal(t, responseMap["title"], v.title)
			assert.Equal(t, responseMap["content"], v.content)
			assert.Equal(t, responseMap["author_id"], float64(v.author_id))
		}
		if v.statusCode == 401 || v.statusCode == 422 || v.statusCode == 500 && v.errorMessage != "" {
			assert.Equal(t, responseMap["error"], v.errorMessage)
		}
	}
}

func TestGetPosts(t *testing.T) {
	err := refreshUserAndPostTable()
	if err != nil {
		log.Fatal(err)
	}

	_, _, err = seedUsersAndPosts()
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("GET", "/posts", nil)
	if err != nil {
		t.Errorf("this is the error: %v\n", err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetPosts)
	handler.ServeHTTP(rr, req)

	var posts []models.Post
	err = json.Unmarshal([]byte(rr.Body.String()), &posts)
	if err != nil {
		t.Errorf("Could not unmarshal: %v\n", err)
	}

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Equal(t, len(posts), 2)
}

func TestGetPostByID(t *testing.T) {
	err := refreshUserAndPostTable()
	if err != nil {
		log.Fatal(err)
	}
	post, err := seedOneUserAndOnePost()
	if err != nil {
		log.Fatal(err)
	}

	postSample := []struct {
		id           string
		statusCode   int
		title        string
		content      string
		author_id    uint32
		errorMessage string
	}{
		{
			id:         strconv.Itoa(int(post.ID)),
			statusCode: 200,
			title:      post.Title,
			content:    post.Content,
			author_id:  post.AuthorID,
		}, {
			id:         "unknown",
			statusCode: 400,
		},
	}

	for _, v := range postSample {
		req, err := http.NewRequest("GET", "/posts", nil)
		if err != nil {
			t.Errorf("this is the error: %v\n", err)
		}
		req = mux.SetURLVars(req, map[string]string{"id": v.id})
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.GetPost)
		handler.ServeHTTP(rr, req)

		responseMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
		if err != nil {
			log.Fatalf("Cannot convert to json: %v", err)
		}
		assert.Equal(t, rr.Code, v.statusCode)
		if v.statusCode == 200 {
			assert.Equal(t, post.Title, responseMap["title"])
			assert.Equal(t, post.Content, responseMap["content"])
			assert.Equal(t, float64(post.AuthorID), responseMap["author_id"])
		}
	}
}

func TestUpdatePost(t *testing.T) {

	var PostUserEmail, PostUserPassword string
	var AuthorPostAuthorID uint32
	var AuthPostID uint64

	err := refreshUserAndPostTable()
	if err != nil {
		log.Fatal(err)
	}
	users, posts, err := seedUsersAndPosts()
	if err != nil {
		log.Fatal(err)
	}

	//Get only the first user
	for _, user := range users {
		if user.ID == 2 {
			continue
		}
		PostUserEmail = user.Email
		PostUserPassword = "password" // password in db is hashed, we do not want hashed one
	}

	token, err := server.SignIn(PostUserEmail, PostUserPassword)
	if err != nil {
		log.Fatalf("cannot login: %v\n", err)
	}

	tokenString := fmt.Sprintf("Bearer %v", token)

	// Get only the first post
	for _, post := range posts {
		if post.ID == 2 {
			continue
		}

		AuthPostID = post.ID
		AuthorPostAuthorID = post.AuthorID
	}

	samples := []struct {
		id           string
		updateJSON   string
		statusCode   int
		title        string
		content      string
		author_id    uint32
		tokenGiven   string
		errorMessage string
	}{
		{
			id:           strconv.Itoa(int(AuthPostID)),
			updateJSON:   `{"title": "The updated post", "content": "This is the updated content", "author_id": 1}`,
			statusCode:   200,
			title:        "The updated post",
			content:      "This is the updated content",
			author_id:    AuthorPostAuthorID,
			tokenGiven:   tokenString,
			errorMessage: "",
		}, {
			// no token
			id:           strconv.Itoa(int(AuthPostID)),
			updateJSON:   `{"title": "This is still another title", "content": "another content", "author_id": 1}`,
			tokenGiven:   "",
			statusCode:   401,
			errorMessage: "Unauthorized",
		}, {
			//wrong token
			id:           strconv.Itoa(int(AuthPostID)),
			updateJSON:   `{"title": "This is another title", "content": "Another content", "author_id": 1}`,
			tokenGiven:   "WRONG TKEN",
			statusCode:   401,
			errorMessage: "Unauthorized",
		}, {
			//title 2 belongs to post 2, and title must be unique
			id:           strconv.Itoa(int(AuthPostID)),
			updateJSON:   `{"title": "Title 2", "content": "another content", "author_id": 1}`,
			tokenGiven:   tokenString,
			statusCode:   500,
			errorMessage: "Title Already Taken",
		}, {
			id: strconv.Itoa(int(AuthPostID)),
			updateJSON: `{"title":"", "content": "Another content", "author_id": 1
			}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Title",
		}, {
			id:           strconv.Itoa(int(AuthPostID)),
			updateJSON:   `{"title": "Awesome title", "content": "", "author_id": 1}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Content",
		}, {
			id:           strconv.Itoa(int(AuthPostID)),
			updateJSON:   `{"title": "This is another title", "content": "This is content"}`,
			statusCode:   401,
			tokenGiven:   tokenString,
			errorMessage: "Unauthorized",
		}, {
			id:         "unwokdn",
			statusCode: 400,
		}, {
			id:           strconv.Itoa(int(AuthPostID)),
			updateJSON:   `{"title": "This is another title", "content": "This is updated content", "author_id": 2}`,
			tokenGiven:   tokenString,
			statusCode:   401,
			errorMessage: "Unauthorized",
		},
	}

	for _, v := range samples {
		req, err := http.NewRequest("POST", "/posts", bytes.NewBufferString(v.updateJSON))
		if err != nil {
			t.Errorf("this is the error: %v\n ", err)
		}

		req = mux.SetURLVars(req, map[string]string{"id": v.id})
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.UpdatePost)

		req.Header.Set("Authorization", v.tokenGiven)

		handler.ServeHTTP(rr, req)

		responseMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
		if err != nil {
			t.Errorf("Cannot convert to json: %v", err)
		}

		assert.Equal(t, rr.Code, v.statusCode)
		if v.statusCode == 200 {
			assert.Equal(t, responseMap["title"], v.title)
			assert.Equal(t, responseMap["content"], v.content)
			assert.Equal(t, responseMap["author_id"], float64(v.author_id))
		}
		if v.statusCode == 401 || v.statusCode == 422 || v.statusCode == 500 && v.errorMessage != "" {
			assert.Equal(t, responseMap["error"], v.errorMessage)
		}
	}
}

func TestDeletePost(t *testing.T) {
	var PostUserEmail, PostUserPassword string
	var PostUserID uint32
	var AuthPostID uint64

	err := refreshUserAndPostTable()
	if err != nil {
		log.Fatal(err)
	}

	users, posts, err := seedUsersAndPosts()
	if err != nil {
		log.Fatal(err)
	}
	// get only 2nd user
	for _, user := range users {
		if user.ID == 1 {
			continue
		}
		PostUserEmail = user.Email
		PostUserPassword = "password"
	}

	// log in
	token, err := server.SignIn(PostUserEmail, PostUserPassword)
	if err != nil {
		log.Fatalf("cannot login: %v\n", err)
	}
	tokenString := fmt.Sprintf("Bearer %v", token)

	//get only the second post
	for _, post := range posts {
		if post.ID == 1 {
			continue
		}
		AuthPostID = post.ID
		PostUserID = post.AuthorID
	}
	postSample := []struct {
		id           string
		author_id    uint32
		tokenGiven   string
		statusCode   int
		errorMessage string
	}{
		{
			id:           strconv.Itoa(int(AuthPostID)),
			author_id:    PostUserID,
			tokenGiven:   tokenString,
			statusCode:   204,
			errorMessage: "",
		}, {
			//no token
			id:           strconv.Itoa(int(AuthPostID)),
			author_id:    PostUserID,
			tokenGiven:   "",
			statusCode:   401,
			errorMessage: "Unauthorized",
		}, {
			// wrokg token
			id:           strconv.Itoa(int(AuthPostID)),
			author_id:    PostUserID,
			tokenGiven:   "This is an incorrect token",
			statusCode:   401,
			errorMessage: "Unauthorized",
		}, {
			id:         "unkwon",
			tokenGiven: tokenString,
			statusCode: 400,
		}, {
			id:           "1",
			author_id:    1,
			statusCode:   401,
			tokenGiven:   tokenString,
			errorMessage: "Unauthorized",
		},
	}
	for _, v := range postSample {
		req, _ := http.NewRequest("GET", "/posts", nil)
		req = mux.SetURLVars(req, map[string]string{"id": v.id})

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.DeletePost)

		req.Header.Set("Authorization", v.tokenGiven)

		handler.ServeHTTP(rr, req)

		assert.Equal(t, rr.Code, v.statusCode)

		if v.statusCode == 401 && v.errorMessage != "" {
			responseMap := make(map[string]interface{})
			err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
			if err != nil {
				t.Errorf("Cannot convert to json: %v", err)
			}
			assert.Equal(t, responseMap["error"], v.errorMessage)
		}
	}

}
