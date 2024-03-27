package main

import (
	"html/template"
	"io"
	"net/http"
	"strconv"

	"github.com/golang-htmx/golang-htmx/cmd/middleware"
)

type Templates struct {
	Templates *template.Template
}

func (t *Templates) Render(w io.Writer, name string, data interface{}) error {
	return t.Templates.ExecuteTemplate(w, name, data)
}

func NewTemplates() *Templates {
	return &Templates{
		Templates: template.Must(template.ParseGlob("views/*.html")),
	}
}

var id = 0

type Contact struct {
	Name  string
	Email string
	ID    int
}

func NewContact(name string, email string) Contact {
	id++
	return Contact{
		Name:  name,
		Email: email,
		ID:    id,
	}
}

type Contacts = []Contact

type Data struct {
	Contacts Contacts
}

func NewData() Data {
	return Data{
		Contacts: []Contact{
			NewContact("a", "a"),
		},
	}
}

func (d *Data) HasEmail(email string) bool {
	for _, contact := range d.Contacts {
		if contact.Email == email {
			return true
		}
	}
	return false
}

func (d *Data) IndexOfId(id int) int {
	for i, contact := range d.Contacts {
		if contact.ID == id {
			return i
		}
	}
	return -1
}

type FormData struct {
	Values map[string]string
	Errors map[string]string
}

func NewFormData() FormData {
	return FormData{
		Values: make(map[string]string),
		Errors: make(map[string]string),
	}
}

type Page struct {
	Data
	FormData
}

func NewPage() Page {
	return Page{
		Data:     NewData(),
		FormData: NewFormData(),
	}
}

func main() {
	mux := http.NewServeMux()

	renderer := NewTemplates()

	page := NewPage()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		err := renderer.Render(w, "index", page)
		if err != nil {
			panic(err)
		}
	})
	mux.HandleFunc("POST /contacts", func(w http.ResponseWriter, r *http.Request) {
		name := r.FormValue("name")
		email := r.FormValue("email")
		if page.HasEmail(email) {
			formData := NewFormData()
			formData.Values["name"] = name
			formData.Values["email"] = email

			formData.Errors["email"] = "There already is a contact with this email"

			err := renderer.Render(w, "form", formData)
			if err != nil {
				panic(err)
			}
			return
		}
		contact := NewContact(name, email)
		page.Contacts = append(page.Contacts, contact)

		err := renderer.Render(w, "form", NewFormData())
		if err != nil {
			panic(err)
		}
		err = renderer.Render(w, "oob-contact", contact)
		if err != nil {
			panic(err)
		}
	})

	mux.HandleFunc("DELETE /contacts/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid id", http.StatusBadRequest)
		}
		index := page.IndexOfId(id)
		if index == -1 {
			http.Error(w, "Invalid id", http.StatusNotFound)
		}
		page.Contacts = append(page.Contacts[:index], page.Contacts[index+1:]...)
	})

	server := http.Server{
		Addr:    ":8080",
		Handler: middleware.Logging(mux),
	}

	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
