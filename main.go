package main

import (
	"PersonalWebsite/connection"
	"PersonalWebsite/middleware"
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

func main() {

	route := mux.NewRouter()

	connection.DatabaseConnect()

	route.PathPrefix("/css/").Handler(http.StripPrefix("/css/", http.FileServer(http.Dir("./css"))))
	route.PathPrefix("/img/").Handler(http.StripPrefix("/img/", http.FileServer(http.Dir("./img"))))
	route.PathPrefix("/icon/").Handler(http.StripPrefix("/icon/", http.FileServer(http.Dir("./icon"))))

	route.HandleFunc("/home", home).Methods("GET")
	route.HandleFunc("/add_myproject", addMyProject).Methods("GET")
	route.HandleFunc("/contact_me", contactMe).Methods("GET")
	route.HandleFunc("/detail_project/{ID}", detailProject).Methods("GET")
	route.HandleFunc("/add_myproject", middleware.UploadFile(ambilData)).Methods("POST")
	route.HandleFunc("/delete_project/{ID}", deleteProject).Methods("GET")
	route.HandleFunc("/halaman_edit/{ID}", halamanEdit).Methods("GET")
	route.HandleFunc("/submit_halaman_edit/{ID}", submitHalamanEdit).Methods("POST")

	route.HandleFunc("/form_register_user", formRegisterUser).Methods("GET")
	route.HandleFunc("/form_register_user", register).Methods("POST")

	route.HandleFunc("/form_login_user", formLoginUser).Methods("GET")
	route.HandleFunc("/login", login).Methods("POST")

	route.HandleFunc("/logout", logout).Methods("GET")

	fmt.Println("server running on port 80")
	http.ListenAndServe("localhost:80", route)

}

type SessionData struct {
	IsLogin   bool
	UserName  string
	FlashData string
}

var Data = SessionData{}

type User struct {
	ID       int
	Name     string
	Email    string
	Password string
}

type Project struct {
	NamaProject string
	Description string
	Start_Date  time.Time
	End_Date    time.Time
	Durasi      string
	Author      string
	Image       string
	IsLogin     bool
	// NodeJS       string
	// ReactJS      string
	// JavaScript   string
	// SocketIO     string
	Format_SDate string
	Format_EDate string
	ID           int
}

func ambilData(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	var projectName = r.PostForm.Get("project-name")
	var description = r.PostForm.Get("description")
	var startdate = r.PostForm.Get("start-date")
	var enddate = r.PostForm.Get("end-date")

	Format := "2006-01-02"
	var sdate, _ = time.Parse(Format, startdate)
	var edate, _ = time.Parse(Format, enddate)
	durasiDalamJam := edate.Sub(sdate).Hours()

	durasiDalamHari := durasiDalamJam / 24
	durasiDalamBulan := durasiDalamHari / 30
	durasiDalamTahun := durasiDalamBulan / 12

	var durasi string
	var hari, _ float64 = math.Modf(durasiDalamHari)
	var bulan, _ float64 = math.Modf(durasiDalamBulan)
	var tahun, _ float64 = math.Modf(durasiDalamTahun)

	if tahun > 0 {
		durasi = "durasi: " + strconv.FormatFloat(tahun, 'f', 0, 64) + " Tahun"
	} else if bulan > 0 {
		durasi = "durasi: " + strconv.FormatFloat(bulan, 'f', 0, 64) + " Bulan"
	} else if hari > 0 {
		durasi = "durasi: " + strconv.FormatFloat(hari, 'f', 0, 64) + " Hari"
	} else if durasiDalamJam > 0 {
		durasi = "durasi: " + strconv.FormatFloat(durasiDalamJam, 'f', 0, 64) + " Jam"
	} else {
		durasi = "durasi: 0 Hari"
	}

	dataContext := r.Context().Value("dataFile")
	image := dataContext.(string)

	var store = sessions.NewCookieStore([]byte("SESSION_KEY"))
	session, _ := store.Get(r, "SESSION_KEY")

	author := session.Values["ID"].(int)
	fmt.Println(author)

	_, err = connection.Conn.Exec(context.Background(), "INSERT INTO tb_projects(name, description, start_date, end_date, durasi, author_id, image) Values($1, $2, $3, $4, $5, $6, $7)", projectName, description, sdate, edate, durasi, author, image)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	http.Redirect(w, r, "/home", http.StatusMovedPermanently)

}

func home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var tmpl, err = template.ParseFiles("html/index.html")

	if err != nil {
		w.Write([]byte("message : " + err.Error()))
		return
	}

	var store = sessions.NewCookieStore([]byte("SESSION_KEY"))
	session, _ := store.Get(r, "SESSION_KEY")

	if session.Values["IsLogin"] != true {
		Data.IsLogin = false
	} else {
		Data.IsLogin = session.Values["IsLogin"].(bool)
		Data.UserName = session.Values["Name"].(string)
	}

	fm := session.Flashes("message")

	var flashes []string
	if len(fm) > 0 {
		session.Save(r, w)
		for _, f1 := range fm {
			// meamasukan flash message
			flashes = append(flashes, f1.(string))
		}
	}

	Data.FlashData = strings.Join(flashes, "")

	data, _ := connection.Conn.Query(context.Background(), "SELECT tb_projects.id, tb_projects.name, description, durasi, image, tb_user.name as author FROM tb_projects LEFT JOIN tb_user ON tb_projects.author_id = tb_user.id ORDER by id DESC")

	var result []Project

	for data.Next() {
		var each = Project{}

		err := data.Scan(&each.ID, &each.NamaProject, &each.Description, &each.Durasi, &each.Image, &each.Author)

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		result = append(result, each)
	}

	resData := map[string]interface{}{
		"Projects":    result,
		"DataSession": Data,
	}

	tmpl.Execute(w, resData)
}

func addMyProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var tmpl, err = template.ParseFiles("html/add_myproject.html")

	if err != nil {
		w.Write([]byte("message : " + err.Error()))
		return
	}

	tmpl.Execute(w, nil)
}

func contactMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var tmpl, err = template.ParseFiles("html/contact_me.html")

	if err != nil {
		w.Write([]byte("message : " + err.Error()))
		return
	}

	tmpl.Execute(w, nil)
}

func detailProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var tmpl, err = template.ParseFiles("html/detail_project.html")

	if err != nil {
		w.Write([]byte("message : " + err.Error()))
		return
	}

	var ProjectDetail = Project{}

	ID, _ := strconv.Atoi(mux.Vars(r)["ID"])

	err = connection.Conn.QueryRow(context.Background(), "SELECT tb_projects.id, name, description, start_date, end_date, durasi, image, tb_user.name as author FROM tb_projects LEFT JOIN tb_user ON tb_projects.author_id = tb_user.id WHERE tb_projects.id = $1", ID).Scan(&ProjectDetail.ID, &ProjectDetail.NamaProject, &ProjectDetail.Description, &ProjectDetail.Start_Date, &ProjectDetail.End_Date, &ProjectDetail.Durasi, &ProjectDetail.Image, &ProjectDetail.Author)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	ProjectDetail.Format_SDate = ProjectDetail.Start_Date.Format("02 January 2006")
	ProjectDetail.Format_EDate = ProjectDetail.End_Date.Format("02 January 2006")
	data := map[string]interface{}{
		"Projects": ProjectDetail,
	}

	tmpl.Execute(w, data)
}

func deleteProject(w http.ResponseWriter, r *http.Request) {
	ID, _ := strconv.Atoi(mux.Vars(r)["ID"])

	_, err := connection.Conn.Exec(context.Background(), "DELETE FROM tb_projects WHERE id = $1", ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	http.Redirect(w, r, "/home", http.StatusFound)
}

func halamanEdit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var tmpl, err = template.ParseFiles("html/halaman_edit.html")

	if err != nil {
		w.Write([]byte("message :" + err.Error()))
		return
	}
	var ProjectDetail = Project{}
	ID, _ := strconv.Atoi(mux.Vars(r)["ID"])

	err = connection.Conn.QueryRow(context.Background(), "SELECT id, name, description FROM tb_projects WHERE id = $1", ID).Scan(&ProjectDetail.ID, &ProjectDetail.NamaProject, &ProjectDetail.Description)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	data := map[string]interface{}{
		"EditProject": ProjectDetail,
	}
	tmpl.Execute(w, data)
}

func submitHalamanEdit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	ID, _ := strconv.Atoi(mux.Vars(r)["ID"])

	var projectName = r.PostForm.Get("project-name")
	var description = r.PostForm.Get("description")
	var startdate = r.PostForm.Get("start-date")
	var enddate = r.PostForm.Get("end-date")

	Format := "2006-01-02"
	var sdate, _ = time.Parse(Format, startdate)
	var edate, _ = time.Parse(Format, enddate)
	durasiDalamJam := edate.Sub(sdate).Hours()

	durasiDalamHari := durasiDalamJam / 24
	durasiDalamBulan := durasiDalamHari / 30
	durasiDalamTahun := durasiDalamBulan / 12

	var durasi string
	var hari, _ float64 = math.Modf(durasiDalamHari)
	var bulan, _ float64 = math.Modf(durasiDalamBulan)
	var tahun, _ float64 = math.Modf(durasiDalamTahun)

	if tahun > 0 {
		durasi = "durasi: " + strconv.FormatFloat(tahun, 'f', 0, 64) + " Tahun"
	} else if bulan > 0 {
		durasi = "durasi: " + strconv.FormatFloat(bulan, 'f', 0, 64) + " Bulan"
	} else if hari > 0 {
		durasi = "durasi: " + strconv.FormatFloat(hari, 'f', 0, 64) + " Hari"
	} else if durasiDalamJam > 0 {
		durasi = "durasi: " + strconv.FormatFloat(durasiDalamJam, 'f', 0, 64) + " Jam"
	} else {
		durasi = "durasi: 0 Hari"
	}

	dataContext := r.Context().Value("dataFile")
	image := dataContext.(string)

	var store = sessions.NewCookieStore([]byte("SESSION_KEY"))
	session, _ := store.Get(r, "SESSION_KEY")

	author := session.Values["ID"].(int)
	fmt.Println(author)

	_, err = connection.Conn.Exec(context.Background(), "UPDATE tb_projects SET name = $1, description = $2, durasi = $3, image = $4, author_id = $5 WHERE id = $6", projectName, description, durasi, image, author, ID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	http.Redirect(w, r, "/home", http.StatusMovedPermanently)

}

func formLoginUser(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var tmpl, err = template.ParseFiles("html/login.html")

	if err != nil {
		w.Write([]byte("message : " + err.Error()))
		return
	}

	var store = sessions.NewCookieStore([]byte("SESSION_KEY"))
	session, _ := store.Get(r, "SESSION_KEY")

	fm := session.Flashes("message")

	var flashes []string
	if len(fm) > 0 {
		session.Save(r, w)
		for _, f1 := range fm {
			// meamasukan flash message
			flashes = append(flashes, f1.(string))
		}
	}

	Data.FlashData = strings.Join(flashes, "")
	tmpl.Execute(w, Data)
}

func formRegisterUser(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var tmpl, err = template.ParseFiles("html/register.html")

	if err != nil {
		w.Write([]byte("message : " + err.Error()))
		return
	}

	tmpl.Execute(w, nil)
}

func register(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	var name = r.PostForm.Get("inputName")
	var email = r.PostForm.Get("inputEmail")
	var password = r.PostForm.Get("inputPassword")

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)

	_, err = connection.Conn.Exec(context.Background(), "INSERT INTO tb_user(name, email, password) VALUES ($1, $2, $3)", name, email, passwordHash)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	http.Redirect(w, r, "/form_register_user", http.StatusMovedPermanently)
}

func login(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	var email = r.PostForm.Get("inputEmail")
	var password = r.PostForm.Get("inputPassword")

	user := User{}

	// mengambil data email, dan melakukan pengecekan email
	err = connection.Conn.QueryRow(context.Background(),
		"SELECT * FROM tb_user WHERE email=$1", email).Scan(&user.ID, &user.Name, &user.Email, &user.Password)

	if err != nil {

		// fmt.Println("Email belum terdaftar")
		var store = sessions.NewCookieStore([]byte("SESSION_KEY"))
		session, _ := store.Get(r, "SESSION_KEY")

		session.AddFlash("Email belum terdaftar!", "message")
		session.Save(r, w)

		http.Redirect(w, r, "/form_login_user", http.StatusMovedPermanently)
		// w.WriteHeader(http.StatusBadRequest)
		// w.Write([]byte("message : Email belum terdaftar " + err.Error()))
		return
	}

	// melakukan pengecekan password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		// fmt.Println("Password salah")
		var store = sessions.NewCookieStore([]byte("SESSION_KEY"))
		session, _ := store.Get(r, "SESSION_KEY")

		session.AddFlash("Password Salah!", "message")
		session.Save(r, w)

		http.Redirect(w, r, "/form_login_user", http.StatusMovedPermanently)
		// w.WriteHeader(http.StatusBadRequest)
		// w.Write([]byte("message : Email belum terdaftar " + err.Error()))
		return
	}

	var store = sessions.NewCookieStore([]byte("SESSION_KEY"))
	session, _ := store.Get(r, "SESSION_KEY")

	// berfungsi untuk menyimpan data kedalam session browser
	session.Values["Name"] = user.Name
	session.Values["Email"] = user.Email
	session.Values["ID"] = user.ID
	session.Values["IsLogin"] = true
	session.Options.MaxAge = 10800 // 3 JAM

	session.AddFlash("succesfull login", "message")
	session.Save(r, w)

	http.Redirect(w, r, "/home", http.StatusMovedPermanently)
}

func logout(w http.ResponseWriter, r *http.Request) {

	var store = sessions.NewCookieStore([]byte("SESSION_KEY"))
	session, _ := store.Get(r, "SESSION_KEY")
	session.Options.MaxAge = -1
	session.Save(r, w)

	http.Redirect(w, r, "/form_login_user", http.StatusSeeOther)
}
