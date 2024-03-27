package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"time"

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
	Name      string
	Email     string
	ID        int
	CreatedAt string
}

func NewContact(name string, email string) Contact {
	id++
	return Contact{
		Name:      name,
		Email:     email,
		ID:        id,
		CreatedAt: time.Now().Format(time.RFC1123),
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

func (d *Data) HasEmail(email string) (bool, int) {
	for _, contact := range d.Contacts {
		if contact.Email == email {
			return true, contact.ID
		}
	}
	return false, -1
}

func (d *Data) IndexOfId(id int) int {
	for i, contact := range d.Contacts {
		if contact.ID == id {
			return i
		}
	}
	return -1
}

func (d *Data) EditContact(old_contact Contact, name string, email string) Contact {
	contact := old_contact
	contact.Email = email
	contact.Name = name
	d.Contacts[d.IndexOfId(contact.ID)] = contact
	return contact
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
	Data     Data
	FormData FormData
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
		found, _ := page.Data.HasEmail(email)
		if found {
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
		page.Data.Contacts = append(page.Data.Contacts, contact)

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
		index := page.Data.IndexOfId(id)
		if index == -1 {
			http.Error(w, "Invalid id", http.StatusNotFound)
		}
		page.Data.Contacts = append(page.Data.Contacts[:index], page.Data.Contacts[index+1:]...)
	})

	mux.HandleFunc("PATCH /contacts/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid id", http.StatusBadRequest)
		}
		index := page.Data.IndexOfId(id)
		if index == -1 {
			http.Error(w, "Invalid id", http.StatusNotFound)
		}
		formData := NewFormData()
		formData.Values["name"] = page.Data.Contacts[index].Name
		formData.Values["email"] = page.Data.Contacts[index].Email
		formData.Values["id"] = fmt.Sprint(id)
		err = renderer.Render(w, "form-change", formData)
		if err != nil {
			panic(err)
		}
	})

	mux.HandleFunc("POST /contacts/edit/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid id", http.StatusBadRequest)
		}
		index := page.Data.IndexOfId(id)
		if index == -1 {
			http.Error(w, "Invalid id", http.StatusNotFound)
		}
		name := r.FormValue("name")
		email := r.FormValue("email")
		found, id_of_found := page.Data.HasEmail(email)
		if found && id_of_found != id {
			formData := NewFormData()
			formData.Values["name"] = name
			formData.Values["email"] = email
			formData.Values["id"] = fmt.Sprint(id)
			formData.Errors["email"] = "There already is a contact with this email"
			err = renderer.Render(w, "form-change", formData)
			if err != nil {
				panic(err)
			}
			return
		}
		contact := page.Data.EditContact(page.Data.Contacts[page.Data.IndexOfId(id)], name, email)
		err = renderer.Render(w, "oob-contact", contact)
		if err != nil {
			panic(err)
		}
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
