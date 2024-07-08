package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
	// "golang.org/x/crypto/bcrypt"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var db *sql.DB

func main() {
	var err error
	connStr := "host=172.17.0.2 port=5432 user=spondon password=1234 dbname=temp_db sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = createTable()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/users", usersHandler)
	http.HandleFunc("/user/", userHandler)

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/loginauth", loginAuthHandler)

	fmt.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<body>

<h2>Login</h2><br>

<form action="/loginauth" method="POST">
    <label for="username">Name:</label><br>
    <input type="text" id="username" name="username"><br>
    <label for="E-mail">E-mail:</label><br>
    <input type="e-mail" id="email" name="email"><br><br>
    <input type="submit" value="Submit">
</form> 

<br><br>
{{if .}}
    {{.}}
{{end}}

</body>
</html>`
	fmt.Fprint(w, html)
}

func loginAuthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("*****loginAuthHandler running*****")
	r.ParseForm()
	var user User
	user.Name = r.FormValue("username")
	user.Email = r.FormValue("email")
	fmt.Println("username:", user.Name, "E-mail:", user.Email)
	// retrieve password from db to compare (hash) with user supplied password's hash
	var hash string
	stmt := "SELECT email FROM users WHERE name = $1"
	row := db.QueryRow(stmt, user.Name)
	fmt.Println(row)
	err := row.Scan(&hash)
	fmt.Println("E-mail from database:", hash)

// 	html := `<!DOCTYPE html>
// <html>
// <body>

// <h2>Login</h2><br>

// <form action="/loginauth" method="POST">
//     <label for="username">Name:</label><br>
//     <input type="text" id="username" name="username"><br>
//     <label for="E-mail">E-mail:</label><br>
//     <input type="e-mail" id="email" name="email"><br><br>
//     <input type="submit" value="Submit">
// </form> 

// <br><br>


// </body>
// </html>`

	if err != nil {
		fmt.Println("error selecting Hash in db by Username")
		fmt.Fprint(w, "A problem occured please go back and try again")
		//return
	}
	// func CompareHashAndPassword(hashedPassword, password []byte) error
	//err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(user.Email))
	// returns nill on succcess
	if hash == user.Email {
		fmt.Fprint(w, "You have successfully logged in :)")
		return
	}
	// if err == nil {
	// 	fmt.Fprint(w, "You have successfully logged in :)")
	// 	return
	// }
	fmt.Fprint(w, "incorrect password")
//	fmt.Fprint(w, html)
}

func createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100),
		email VARCHAR(100)
	);`
	_, err := db.Exec(query)
	return err
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getUsers(w)
	case "POST":
		createUser(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/user/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		getUser(w, id)
	case "PUT":
		updateUser(w, r, id)
	case "DELETE":
		deleteUser(w, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getUsers(w http.ResponseWriter) {
	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if user.Name == "" {
		http.Error(w, "Please mention the name", http.StatusBadRequest)
		return
	}

	if user.Email == "" {
		http.Error(w, "Please mention the e-mail", http.StatusBadRequest)
		return
	}

	err := db.QueryRow("INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id", user.Name, user.Email).Scan(&user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func getUser(w http.ResponseWriter, id int) {
	var user User
	err := db.QueryRow("SELECT id, name, email FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name, &user.Email)
	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func updateUser(w http.ResponseWriter, r *http.Request, id int) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := "UPDATE users SET"
	params := []interface{}{}
	paramID := 1

	if user.Name != "" {
		query += " name = $" + strconv.Itoa(paramID)
		params = append(params, user.Name)
		paramID++
	}

	if user.Email != "" {
		if paramID > 1 {
			query += ","
		}
		query += " email = $" + strconv.Itoa(paramID)
		params = append(params, user.Email)
		paramID++
	}

	if paramID == 1 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	query += " WHERE id = $" + strconv.Itoa(paramID)
	params = append(params, id)

	_, err := db.Exec(query, params...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// _, err := db.Exec("UPDATE users SET name = $1, email = $2 WHERE id = $3", user.Name, user.Email, id)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	w.WriteHeader(http.StatusNoContent)
}

func deleteUser(w http.ResponseWriter, id int) {
	_, err := db.Exec("DELETE FROM users WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		http.Error(w, "User deleted", http.StatusNotFound)
	}

	w.WriteHeader(http.StatusNoContent)
}
